package persistence

import (
	"bytes"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/operations"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func (store *Store) Create(
	ctx context.Context,
	command operations.CreateApprovalCommand,
) (operations.ApprovalResult, error) {
	for range 3 {
		result, err := store.createOnce(ctx, command)
		if !errors.Is(err, errRetryApprovalTransaction) {
			return result, err
		}
	}
	return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
}

func (store *Store) createOnce(
	ctx context.Context,
	command operations.CreateApprovalCommand,
) (operations.ApprovalResult, error) {
	if command.Actor.SessionID.IsZero() || command.Actor.PrincipalID.IsZero() ||
		command.ApprovalID.IsZero() || command.ApprovalID.Prefix() != "apr" ||
		command.ActionType != operations.ActionMembershipChangeAdmin ||
		len(command.CanonicalPayload) < 64 || command.PayloadDigest == ([32]byte{}) ||
		command.Now.IsZero() || command.ExpiresAt.IsZero() || !command.ExpiresAt.After(command.Now) {
		return operations.ApprovalResult{}, operations.ErrApprovalInputInvalid
	}
	transaction, err := beginApprovalTransaction(ctx, store.pool)
	if err != nil {
		return operations.ApprovalResult{}, err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	authorization, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: command.Payload.OrganizationID,
			Action: identity.ApprovalActionCreate, AdditionalAction: identity.ApprovedRoleChangeAction,
			Purpose: "organization_administration", Resource: "membership",
			ResourceVersion: command.Payload.ExpectedMembershipVersion, ResourceStatus: "active",
			RequiredStepUpAction: identity.ApprovalStepUpCreateAction, Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if denial := mapIdentityAuthorization(authorization); denial != nil {
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, authorization.Reason, denial)
	}

	replayed, storedRequest, decisionID, err := loadCreateReplay(ctx, transaction, command)
	if err == nil {
		result := operations.ApprovalResult{
			Approval:   effectiveApproval(replayed, command.Now),
			DecisionID: decisionID, Replay: true,
		}
		if !bytes.Equal(storedRequest, command.RequestDigest[:]) {
			return result, store.commitDenial(
				ctx, transaction, command.AuditEvent, "idempotency_conflict",
				operations.ErrApprovalIdempotencyConflict,
			)
		}
		if err := commitApprovalTransaction(ctx, transaction); err != nil {
			return operations.ApprovalResult{}, err
		}
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}

	target, err := store.identity.ValidateMembershipRoleChangeTarget(ctx, transaction,
		identity.MembershipRoleChangeTargetRequest{
			OrganizationID:  command.Payload.OrganizationID,
			MembershipID:    command.Payload.MembershipID,
			ExpectedVersion: command.Payload.ExpectedMembershipVersion,
			RequestedRole:   command.Payload.RequestedRole,
		})
	if err != nil {
		denial, reason := mapTargetValidation(err)
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, reason, denial)
	}
	if target.OrganizationID != command.Payload.OrganizationID ||
		target.MembershipID != command.Payload.MembershipID ||
		target.Version != command.Payload.ExpectedMembershipVersion ||
		target.RequestedRole != command.Payload.RequestedRole {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}

	stored, err := scanStoredApproval(transaction.QueryRow(ctx, `
INSERT INTO atlas_operations.approvals (
    approval_id, tenant_id, action_type, target_type, target_id, target_version,
    payload_canonical, payload_sha256, payload_schema_version,
    payload_canonicalization, payload_hash_algorithm,
    requester_principal_id, requester_session_id, eligible_checker_policy,
    purpose, status, version, expires_at, created_at, updated_at
) VALUES (
    $1, $2, $3, 'membership', $4, $5,
    $6, $7, 1,
    'rfc8785', 'sha256',
    $8, $9, 'dynamic-at-decision-and-execution',
    'organization_administration', 'pending', 1, $10, $11, $11
)
RETURNING `+approvalColumns,
		command.ApprovalID.String(), command.Payload.OrganizationID.String(), command.ActionType,
		command.Payload.MembershipID.String(), command.Payload.ExpectedMembershipVersion,
		command.CanonicalPayload, command.PayloadDigest[:], command.Actor.PrincipalID.String(),
		command.Actor.SessionID.String(), command.ExpiresAt, command.Now,
	))
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	inserted, err := transaction.Exec(ctx, `
INSERT INTO atlas_operations.approval_requests (
    approval_id, tenant_id, requester_principal_id, idempotency_key_sha256,
    request_sha256, authorization_decision_id, audit_event_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (tenant_id, requester_principal_id, idempotency_key_sha256) DO NOTHING`,
		command.ApprovalID.String(), command.Payload.OrganizationID.String(),
		command.Actor.PrincipalID.String(), command.IdempotencyDigest[:], command.RequestDigest[:],
		command.AuditEvent.DecisionID.String(), command.AuditEvent.AuditEventID.String(), command.Now,
	)
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	if inserted.RowsAffected() != 1 {
		return operations.ApprovalResult{}, errRetryApprovalTransaction
	}
	event := command.AuditEvent
	event.SafeAfterReference = "approval:pending"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if err := commitApprovalTransaction(ctx, transaction); err != nil {
		return operations.ApprovalResult{}, err
	}
	return operations.ApprovalResult{
		Approval: effectiveApproval(stored, command.Now), DecisionID: event.DecisionID,
	}, nil
}

func loadCreateReplay(
	ctx context.Context,
	transaction pgx.Tx,
	command operations.CreateApprovalCommand,
) (storedApproval, []byte, identifier.ID, error) {
	var approvalIDText, decisionIDText string
	var requestDigest []byte
	err := transaction.QueryRow(ctx, `
SELECT approval_id, request_sha256, authorization_decision_id
FROM atlas_operations.approval_requests
WHERE tenant_id = $1
  AND requester_principal_id = $2
  AND idempotency_key_sha256 = $3`,
		command.Payload.OrganizationID.String(), command.Actor.PrincipalID.String(),
		command.IdempotencyDigest[:],
	).Scan(&approvalIDText, &requestDigest, &decisionIDText)
	if err != nil {
		return storedApproval{}, nil, identifier.ID{}, err
	}
	approvalID, err := parseApprovalIdentifier(approvalIDText, "apr")
	if err != nil {
		return storedApproval{}, nil, identifier.ID{}, operations.ErrApprovalUnavailable
	}
	decisionID, err := parseApprovalIdentifier(decisionIDText, "dec")
	if err != nil {
		return storedApproval{}, nil, identifier.ID{}, operations.ErrApprovalUnavailable
	}
	stored, err := loadApprovalForTenant(ctx, transaction, command.Payload.OrganizationID, approvalID, false)
	return stored, requestDigest, decisionID, err
}

func (store *Store) List(
	ctx context.Context,
	command operations.ListApprovalsCommand,
) (operations.ApprovalPage, error) {
	transaction, err := beginApprovalTransaction(ctx, store.pool)
	if err != nil {
		return operations.ApprovalPage{}, err
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	authorization, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: command.Actor.TenantID,
			Action: identity.ApprovalActionRead, Purpose: "self_service", Resource: "approval",
			ResourceVersion: 1, ResourceStatus: "pending", Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalPage{}, operations.ErrApprovalUnavailable
	}
	if denial := mapIdentityAuthorization(authorization); denial != nil {
		return operations.ApprovalPage{}, store.commitDenial(
			ctx, transaction, command.AuditEvent, authorization.Reason, denial,
		)
	}
	pageSize, err := parseApprovalPage(command.PageSize, command.PageProvided)
	if err != nil {
		return operations.ApprovalPage{}, err
	}
	cursorTime, cursorID, err := decodeApprovalCursor(command.Cursor, command.CursorProvided)
	if err != nil {
		return operations.ApprovalPage{}, err
	}
	var rows pgx.Rows
	if command.CursorProvided {
		rows, err = transaction.Query(ctx, `
SELECT `+approvalColumns+`
FROM atlas_operations.approvals
WHERE tenant_id = $1
  AND (created_at < $2 OR (created_at = $2 AND approval_id < $3))
ORDER BY created_at DESC, approval_id DESC
LIMIT $4`, command.Actor.TenantID.String(), cursorTime, cursorID.String(), pageSize+1)
	} else {
		rows, err = transaction.Query(ctx, `
SELECT `+approvalColumns+`
FROM atlas_operations.approvals
WHERE tenant_id = $1
ORDER BY created_at DESC, approval_id DESC
LIMIT $2`, command.Actor.TenantID.String(), pageSize+1)
	}
	if err != nil {
		return operations.ApprovalPage{}, operations.ErrApprovalUnavailable
	}
	defer rows.Close()
	stored := make([]storedApproval, 0, pageSize+1)
	for rows.Next() {
		approval, scanErr := scanStoredApproval(rows)
		if scanErr != nil {
			return operations.ApprovalPage{}, operations.ErrApprovalUnavailable
		}
		stored = append(stored, approval)
	}
	if rows.Err() != nil {
		return operations.ApprovalPage{}, operations.ErrApprovalUnavailable
	}
	page := operations.ApprovalPage{Approvals: make([]operations.Approval, 0, pageSize)}
	if len(stored) > pageSize {
		stored = stored[:pageSize]
		page.HasMore = true
	}
	for _, item := range stored {
		page.Approvals = append(page.Approvals, effectiveApproval(item, command.Now))
	}
	if page.HasMore && len(stored) > 0 {
		last := stored[len(stored)-1].approval
		page.NextCursor, err = encodeApprovalCursor(last.CreatedAt, last.ApprovalID)
		if err != nil {
			return operations.ApprovalPage{}, err
		}
	}
	event := command.AuditEvent
	event.Decision = "allowed"
	event.SafeAfterReference = "approvals:tenant-page"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return operations.ApprovalPage{}, operations.ErrApprovalUnavailable
	}
	if err := commitApprovalTransaction(ctx, transaction); err != nil {
		return operations.ApprovalPage{}, err
	}
	return page, nil
}

func (store *Store) Get(
	ctx context.Context,
	command operations.GetApprovalCommand,
) (operations.ApprovalResult, error) {
	transaction, err := beginApprovalTransaction(ctx, store.pool)
	if err != nil {
		return operations.ApprovalResult{}, err
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	preliminary, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: command.Actor.TenantID,
			Action: identity.ApprovalActionRead, Purpose: "self_service", Resource: "approval",
			ResourceVersion: 1, ResourceStatus: "pending", Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if denial := mapIdentityAuthorization(preliminary); denial != nil {
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, preliminary.Reason, denial)
	}
	stored, err := loadApprovalForTenant(
		ctx, transaction, command.Actor.TenantID, command.ApprovalID, false,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, "not_found_or_concealed", operations.ErrApprovalNotFound)
	}
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	approval := effectiveApproval(stored, command.Now)
	actual, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: approval.OrganizationID,
			Action: identity.ApprovalActionRead, Purpose: "self_service", Resource: "approval",
			ResourceVersion: approval.Version, ResourceStatus: string(approval.Status), Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if denial := mapIdentityAuthorization(actual); denial != nil {
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, actual.Reason, denial)
	}
	event := command.AuditEvent
	event.Decision = "allowed"
	event.SafeAfterReference = "approval:" + string(approval.Status)
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if err := commitApprovalTransaction(ctx, transaction); err != nil {
		return operations.ApprovalResult{}, err
	}
	return operations.ApprovalResult{
		Approval: approval, DecisionID: event.DecisionID,
	}, nil
}

func loadApprovalForTenant(
	ctx context.Context,
	transaction pgx.Tx,
	tenantID, approvalID identifier.ID,
	lock bool,
) (storedApproval, error) {
	query := `SELECT ` + approvalColumns + `
FROM atlas_operations.approvals
WHERE tenant_id = $1 AND approval_id = $2`
	if lock {
		query += " FOR UPDATE"
	}
	return scanStoredApproval(transaction.QueryRow(ctx, query, tenantID.String(), approvalID.String()))
}

func mapTargetValidation(err error) (error, string) {
	switch {
	case errors.Is(err, identity.ErrMembershipNotFound):
		return operations.ErrApprovalConflict, "target_not_found_or_inactive"
	case errors.Is(err, identity.ErrMembershipPreconditionFailed):
		return operations.ErrApprovalConflict, "target_version_mismatch"
	case errors.Is(err, identity.ErrValidationFailed):
		return operations.ErrApprovalValidationFailed, "typed_action_invalid"
	default:
		return operations.ErrApprovalUnavailable, "target_unavailable"
	}
}
