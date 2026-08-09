package persistence

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/operations"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func (store *Store) Decide(
	ctx context.Context,
	command operations.DecideApprovalCommand,
) (operations.ApprovalResult, error) {
	for range 3 {
		result, err := store.decideOnce(ctx, command)
		if !errors.Is(err, errRetryApprovalTransaction) {
			return result, err
		}
	}
	return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
}

func (store *Store) decideOnce(
	ctx context.Context,
	command operations.DecideApprovalCommand,
) (operations.ApprovalResult, error) {
	if command.Actor.SessionID.IsZero() || command.Actor.PrincipalID.IsZero() ||
		command.DecisionRecordID.IsZero() || command.DecisionRecordID.Prefix() != "apd" ||
		command.ApprovalID.IsZero() || command.ApprovalID.Prefix() != "apr" ||
		command.ExpectedVersion < 1 || command.Purpose != "approval_review" ||
		(command.Decision != "approve" && command.Decision != "reject") ||
		command.Now.IsZero() {
		return operations.ApprovalResult{}, operations.ErrApprovalInputInvalid
	}
	transaction, err := beginApprovalTransaction(ctx, store.pool)
	if err != nil {
		return operations.ApprovalResult{}, err
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	preliminary, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: command.Actor.TenantID,
			Action: identity.ApprovalActionDecide, Purpose: command.Purpose,
			Resource: "approval", ResourceVersion: 1, ResourceStatus: "pending",
			RequiredStepUpAction: identity.ApprovalStepUpDecideAction, Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	if denial := mapIdentityAuthorization(preliminary); denial != nil {
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, preliminary.Reason, denial)
	}

	if replayed, storedRequest, decisionID, replayErr := loadDecisionReplay(ctx, transaction, command); replayErr == nil {
		result := operations.ApprovalResult{
			Approval: effectiveApproval(replayed, command.Now), DecisionID: decisionID, Replay: true,
		}
		if !bytes.Equal(storedRequest, command.RequestDigest[:]) {
			return result, store.commitDenial(ctx, transaction, command.AuditEvent,
				"idempotency_conflict", operations.ErrApprovalIdempotencyConflict)
		}
		if err := commitApprovalTransaction(ctx, transaction); err != nil {
			return operations.ApprovalResult{}, err
		}
		return result, nil
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}

	stored, err := loadApprovalForTenant(ctx, transaction, command.Actor.TenantID, command.ApprovalID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"not_found_or_concealed", operations.ErrApprovalNotFound)
	}
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	approval := effectiveApproval(stored, command.Now)
	if approval.Status == operations.ApprovalExpired {
		return store.expireApproval(ctx, transaction, stored, command.Now, command.AuditEvent)
	}
	if approval.Status != operations.ApprovalPending {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"approval_state_conflict", operations.ErrApprovalConflict)
	}
	if approval.Version != command.ExpectedVersion {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"approval_version_mismatch", operations.ErrApprovalPreconditionFailed)
	}
	actual, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: approval.OrganizationID,
			Action: identity.ApprovalActionDecide, Purpose: command.Purpose,
			Resource: "approval", ResourceVersion: approval.Version,
			ResourceStatus:       string(approval.Status),
			RequiredStepUpAction: identity.ApprovalStepUpDecideAction, Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if denial := mapIdentityAuthorization(actual); denial != nil {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, actual.Reason, denial)
	}
	if approval.RequesterPrincipalID == command.Actor.PrincipalID {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"maker_checker_same_principal", operations.ErrApprovalNotAuthorized)
	}
	status := operations.ApprovalApproved
	reasonCode := "approval_approved"
	if command.Decision == "reject" {
		status = operations.ApprovalRejected
		reasonCode = "approval_rejected"
	}
	updated, err := scanStoredApproval(transaction.QueryRow(ctx, `
UPDATE atlas_operations.approvals
SET status = $3,
    decision_reason = NULLIF($4, ''),
    decider_principal_id = $5,
    decider_session_id = $6,
    version = version + 1,
    decided_at = $7,
    updated_at = $7
WHERE tenant_id = $1
  AND approval_id = $2
  AND status = 'pending'
  AND version = $8
RETURNING `+approvalColumns,
		approval.OrganizationID.String(), approval.ApprovalID.String(), string(status), command.Reason,
		command.Actor.PrincipalID.String(), command.Actor.SessionID.String(), command.Now,
		command.ExpectedVersion,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return operations.ApprovalResult{}, errRetryApprovalTransaction
	}
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	inserted, err := transaction.Exec(ctx, `
INSERT INTO atlas_operations.approval_decisions (
    decision_record_id, approval_id, tenant_id, decider_principal_id,
    decider_session_id, idempotency_key_sha256, request_sha256,
    decision, reason, result_approval_version, authorization_decision_id,
    audit_event_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), $10, $11, $12, $13)
ON CONFLICT (tenant_id, decider_principal_id, idempotency_key_sha256) DO NOTHING`,
		command.DecisionRecordID.String(), approval.ApprovalID.String(), approval.OrganizationID.String(),
		command.Actor.PrincipalID.String(), command.Actor.SessionID.String(),
		command.IdempotencyDigest[:], command.RequestDigest[:], command.Decision, command.Reason,
		updated.approval.Version, command.AuditEvent.DecisionID.String(),
		command.AuditEvent.AuditEventID.String(), command.Now,
	)
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	if inserted.RowsAffected() != 1 {
		return operations.ApprovalResult{}, errRetryApprovalTransaction
	}
	event := command.AuditEvent
	event.ReasonCode = reasonCode
	event.SafeBeforeReference = "approval:pending"
	event.SafeAfterReference = "approval:" + string(status)
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if err := commitApprovalTransaction(ctx, transaction); err != nil {
		return operations.ApprovalResult{}, err
	}
	return operations.ApprovalResult{
		Approval: effectiveApproval(updated, command.Now), DecisionID: event.DecisionID,
	}, nil
}

func (store *Store) Cancel(
	ctx context.Context,
	command operations.CancelApprovalCommand,
) (operations.ApprovalResult, error) {
	for range 3 {
		result, err := store.cancelOnce(ctx, command)
		if !errors.Is(err, errRetryApprovalTransaction) {
			return result, err
		}
	}
	return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
}

func (store *Store) cancelOnce(
	ctx context.Context,
	command operations.CancelApprovalCommand,
) (operations.ApprovalResult, error) {
	if command.Actor.SessionID.IsZero() || command.Actor.PrincipalID.IsZero() ||
		command.CancellationRecordID.IsZero() || command.CancellationRecordID.Prefix() != "apc" ||
		command.ApprovalID.IsZero() || command.ApprovalID.Prefix() != "apr" ||
		command.ExpectedVersion < 1 || len(command.Reason) < 1 || len(command.Reason) > 500 ||
		command.Now.IsZero() {
		return operations.ApprovalResult{}, operations.ErrApprovalInputInvalid
	}
	transaction, err := beginApprovalTransaction(ctx, store.pool)
	if err != nil {
		return operations.ApprovalResult{}, err
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	preliminary, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: command.Actor.TenantID,
			Action: identity.ApprovalActionCancel, Purpose: "organization_administration",
			Resource: "approval", ResourceVersion: 1, ResourceStatus: "pending", Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if denial := mapIdentityAuthorization(preliminary); denial != nil {
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, preliminary.Reason, denial)
	}
	if replayed, storedRequest, decisionID, replayErr := loadCancellationReplay(ctx, transaction, command); replayErr == nil {
		result := operations.ApprovalResult{
			Approval: effectiveApproval(replayed, command.Now), DecisionID: decisionID, Replay: true,
		}
		if !bytes.Equal(storedRequest, command.RequestDigest[:]) {
			return result, store.commitDenial(ctx, transaction, command.AuditEvent,
				"idempotency_conflict", operations.ErrApprovalIdempotencyConflict)
		}
		if err := commitApprovalTransaction(ctx, transaction); err != nil {
			return operations.ApprovalResult{}, err
		}
		return result, nil
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	stored, err := loadApprovalForTenant(ctx, transaction, command.Actor.TenantID, command.ApprovalID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"not_found_or_concealed", operations.ErrApprovalNotFound)
	}
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	approval := effectiveApproval(stored, command.Now)
	if approval.Status == operations.ApprovalExpired {
		return store.expireApproval(ctx, transaction, stored, command.Now, command.AuditEvent)
	}
	actual, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: approval.OrganizationID,
			Action: identity.ApprovalActionCancel, Purpose: "organization_administration",
			Resource: "approval", ResourceVersion: approval.Version,
			ResourceStatus: string(approval.Status), Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if denial := mapIdentityAuthorization(actual); denial != nil {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, actual.Reason, denial)
	}
	if approval.Status != operations.ApprovalPending {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"approval_state_conflict", operations.ErrApprovalConflict)
	}
	if approval.Version != command.ExpectedVersion {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"approval_version_mismatch", operations.ErrApprovalPreconditionFailed)
	}
	updated, err := scanStoredApproval(transaction.QueryRow(ctx, `
UPDATE atlas_operations.approvals
SET status = 'cancelled',
    decision_reason = $3,
    version = version + 1,
    updated_at = $4
WHERE tenant_id = $1 AND approval_id = $2 AND status = 'pending' AND version = $5
RETURNING `+approvalColumns,
		approval.OrganizationID.String(), approval.ApprovalID.String(), command.Reason,
		command.Now, command.ExpectedVersion,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return operations.ApprovalResult{}, errRetryApprovalTransaction
	}
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	inserted, err := transaction.Exec(ctx, `
INSERT INTO atlas_operations.approval_cancellations (
    cancellation_record_id, approval_id, tenant_id, actor_principal_id,
    actor_session_id, idempotency_key_sha256, request_sha256, reason,
    result_approval_version, authorization_decision_id, audit_event_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (tenant_id, actor_principal_id, idempotency_key_sha256) DO NOTHING`,
		command.CancellationRecordID.String(), approval.ApprovalID.String(),
		approval.OrganizationID.String(), command.Actor.PrincipalID.String(),
		command.Actor.SessionID.String(), command.IdempotencyDigest[:], command.RequestDigest[:],
		command.Reason, updated.approval.Version, command.AuditEvent.DecisionID.String(),
		command.AuditEvent.AuditEventID.String(), command.Now,
	)
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	if inserted.RowsAffected() != 1 {
		return operations.ApprovalResult{}, errRetryApprovalTransaction
	}
	event := command.AuditEvent
	event.SafeBeforeReference = "approval:pending"
	event.SafeAfterReference = "approval:cancelled"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if err := commitApprovalTransaction(ctx, transaction); err != nil {
		return operations.ApprovalResult{}, err
	}
	return operations.ApprovalResult{
		Approval: effectiveApproval(updated, command.Now), DecisionID: event.DecisionID,
	}, nil
}

func loadDecisionReplay(
	ctx context.Context,
	transaction pgx.Tx,
	command operations.DecideApprovalCommand,
) (storedApproval, []byte, identifier.ID, error) {
	var approvalIDText, decisionIDText string
	var requestDigest []byte
	err := transaction.QueryRow(ctx, `
SELECT approval_id, request_sha256, authorization_decision_id
FROM atlas_operations.approval_decisions
WHERE tenant_id = $1
  AND decider_principal_id = $2
  AND idempotency_key_sha256 = $3`, command.Actor.TenantID.String(),
		command.Actor.PrincipalID.String(), command.IdempotencyDigest[:],
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
	stored, err := loadApprovalForTenant(ctx, transaction, command.Actor.TenantID, approvalID, false)
	return stored, requestDigest, decisionID, err
}

func loadCancellationReplay(
	ctx context.Context,
	transaction pgx.Tx,
	command operations.CancelApprovalCommand,
) (storedApproval, []byte, identifier.ID, error) {
	var approvalIDText, decisionIDText string
	var requestDigest []byte
	err := transaction.QueryRow(ctx, `
SELECT approval_id, request_sha256, authorization_decision_id
FROM atlas_operations.approval_cancellations
WHERE tenant_id = $1
  AND actor_principal_id = $2
  AND idempotency_key_sha256 = $3`, command.Actor.TenantID.String(),
		command.Actor.PrincipalID.String(), command.IdempotencyDigest[:],
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
	stored, err := loadApprovalForTenant(ctx, transaction, command.Actor.TenantID, approvalID, false)
	return stored, requestDigest, decisionID, err
}

func (store *Store) expireApproval(
	ctx context.Context,
	transaction pgx.Tx,
	stored storedApproval,
	now time.Time,
	event audit.Event,
) (operations.ApprovalResult, error) {
	updated, err := scanStoredApproval(transaction.QueryRow(ctx, `
UPDATE atlas_operations.approvals
SET status = 'expired', version = version + 1, updated_at = $3
WHERE tenant_id = $1 AND approval_id = $2
  AND status IN ('pending', 'approved', 'execution_failed')
RETURNING `+approvalColumns,
		stored.approval.OrganizationID.String(), stored.approval.ApprovalID.String(), now,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return operations.ApprovalResult{}, errRetryApprovalTransaction
	}
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	event.Decision = "denied"
	event.ReasonCode = "approval_expired"
	event.SafeBeforeReference = "approval:" + string(stored.approval.Status)
	event.SafeAfterReference = "approval:expired"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if err := commitApprovalTransaction(ctx, transaction); err != nil {
		return operations.ApprovalResult{}, err
	}
	return operations.ApprovalResult{
		Approval: effectiveApproval(updated, now), DecisionID: event.DecisionID,
	}, operations.ErrApprovalConflict
}
