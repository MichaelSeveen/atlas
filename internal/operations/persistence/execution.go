package persistence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/operations"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func (store *Store) Execute(
	ctx context.Context,
	command operations.ExecuteApprovalCommand,
) (operations.ApprovalResult, error) {
	for range 3 {
		result, err := store.executeOnce(ctx, command)
		if !errors.Is(err, errRetryApprovalTransaction) {
			return result, err
		}
	}
	return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
}

func (store *Store) executeOnce(
	ctx context.Context,
	command operations.ExecuteApprovalCommand,
) (operations.ApprovalResult, error) {
	if command.Actor.SessionID.IsZero() || command.Actor.PrincipalID.IsZero() ||
		command.ExecutionRecordID.IsZero() || command.ExecutionRecordID.Prefix() != "aex" ||
		command.RoleChangeID.IsZero() || command.RoleChangeID.Prefix() != "mrc" ||
		command.ApprovalID.IsZero() || command.ApprovalID.Prefix() != "apr" ||
		command.ExpectedVersion < 1 || command.Now.IsZero() {
		return operations.ApprovalResult{}, operations.ErrApprovalInputInvalid
	}
	transaction, err := beginApprovalTransaction(ctx, store.pool)
	if err != nil {
		return operations.ApprovalResult{}, err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	// A durable response-loss replay needs only current tenant read authority;
	// no target effect or approval transition is repeated.
	preliminary, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: command.Actor.TenantID,
			Action: identity.ApprovalActionRead, Purpose: "organization_administration",
			Resource: "approval", ResourceVersion: 1, ResourceStatus: "pending", Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	if denial := mapIdentityAuthorization(preliminary); denial != nil {
		return operations.ApprovalResult{DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, preliminary.Reason, denial)
	}
	if replayed, storedRequest, decisionID, replayErr := loadExecutionReplay(ctx, transaction, command); replayErr == nil {
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
	if approval.Status != operations.ApprovalApproved && approval.Status != operations.ApprovalExecutionFailed {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"approval_state_conflict", operations.ErrApprovalConflict)
	}
	if approval.Version != command.ExpectedVersion {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"approval_version_mismatch", operations.ErrApprovalPreconditionFailed)
	}
	executorAuthorization, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: approval.OrganizationID,
			Action: identity.ApprovalActionExecute, Purpose: "organization_administration",
			Resource: "approval", ResourceVersion: approval.Version,
			ResourceStatus:       string(approval.Status),
			RequiredStepUpAction: identity.ApprovalStepUpExecuteAction, Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if denial := mapIdentityAuthorization(executorAuthorization); denial != nil {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent, executorAuthorization.Reason, denial)
	}

	payload, integrityReason := validateStoredPayload(stored)
	if integrityReason != "" {
		return store.supersedeApproval(
			ctx, transaction, stored, command.Now, command.AuditEvent, integrityReason,
		)
	}
	if payload.OrganizationID != approval.OrganizationID || payload.MembershipID != stored.targetID ||
		payload.ExpectedMembershipVersion != stored.targetVersion ||
		payload.Purpose != approval.Purpose || approval.ActionType != operations.ActionMembershipChangeAdmin {
		return store.supersedeApproval(
			ctx, transaction, stored, command.Now, command.AuditEvent, "payload_binding_mismatch",
		)
	}
	targetRequest := identity.MembershipRoleChangeTargetRequest{
		OrganizationID: payload.OrganizationID, MembershipID: payload.MembershipID,
		ExpectedVersion: payload.ExpectedMembershipVersion, RequestedRole: payload.RequestedRole,
	}
	target, err := store.identity.ValidateMembershipRoleChangeTarget(ctx, transaction, targetRequest)
	if err != nil {
		reason := "target_state_or_version_changed"
		if errors.Is(err, identity.ErrValidationFailed) {
			reason = "typed_action_no_longer_valid"
		}
		if errors.Is(err, identity.ErrMembershipNotFound) ||
			errors.Is(err, identity.ErrMembershipPreconditionFailed) ||
			errors.Is(err, identity.ErrValidationFailed) {
			return store.supersedeApproval(ctx, transaction, stored, command.Now, command.AuditEvent, reason)
		}
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	makerAuthorization, err := store.identity.AuthorizeApprovalPrincipal(ctx, transaction,
		identity.ApprovalPrincipalAuthorizationRequest{
			PrincipalID: approval.RequesterPrincipalID, OrganizationID: approval.OrganizationID,
			Action: identity.ApprovedRoleChangeAction, Purpose: "organization_administration",
			Resource: "membership", ResourceVersion: target.Version, ResourceStatus: "active",
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if makerAuthorization.Effect != identity.AuthorizationAllow {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"maker_permission_revoked", operations.ErrApprovalNotAuthorized)
	}
	if approval.DeciderPrincipalID.IsZero() || approval.DeciderPrincipalID == approval.RequesterPrincipalID {
		return store.supersedeApproval(
			ctx, transaction, stored, command.Now, command.AuditEvent, "maker_checker_integrity_failure",
		)
	}
	checkerAuthorization, err := store.identity.AuthorizeApprovalPrincipal(ctx, transaction,
		identity.ApprovalPrincipalAuthorizationRequest{
			PrincipalID: approval.DeciderPrincipalID, OrganizationID: approval.OrganizationID,
			Action: identity.ApprovalCheckerEligibilityAction, Purpose: "approval_review",
			Resource: "approval", ResourceVersion: approval.Version,
			ResourceStatus: string(approval.Status),
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if checkerAuthorization.Effect != identity.AuthorizationAllow {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"checker_permission_revoked", operations.ErrApprovalNotAuthorized)
	}
	targetAuthorization, err := store.identity.AuthorizeApprovalSession(ctx, transaction,
		identity.ApprovalSessionAuthorizationRequest{
			Actor: command.Actor, OrganizationID: approval.OrganizationID,
			Action: identity.ApprovedRoleChangeAction, Purpose: "organization_administration",
			Resource: "membership", ResourceVersion: target.Version, ResourceStatus: "active",
			Now: command.Now,
		})
	if err != nil {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if denial := mapIdentityAuthorization(targetAuthorization); denial != nil {
		return operations.ApprovalResult{Approval: approval, DecisionID: command.AuditEvent.DecisionID},
			store.commitDenial(ctx, transaction, command.AuditEvent,
				"executor_target_permission_denied", denial)
	}

	targetEvent := command.TargetAuditEvent
	targetEvent.TargetID = target.MembershipID.String()
	targetEvent.ApprovalID = approval.ApprovalID
	targetResult, err := store.identity.ExecuteApprovedMembershipRoleChange(ctx, transaction,
		identity.ExecuteApprovedMembershipRoleChangeCommand{
			ApprovalID: approval.ApprovalID, RoleChangeID: command.RoleChangeID,
			Executor: command.Actor, Target: targetRequest, PayloadDigest: approval.PayloadDigest,
			IdempotencyDigest: command.IdempotencyDigest, RequestDigest: command.RequestDigest,
			Now: command.Now, AuditEvent: targetEvent,
		})
	if err != nil {
		if errors.Is(err, identity.ErrMembershipNotFound) ||
			errors.Is(err, identity.ErrMembershipPreconditionFailed) ||
			errors.Is(err, identity.ErrValidationFailed) {
			return store.supersedeApproval(
				ctx, transaction, stored, command.Now, command.AuditEvent,
				"target_state_or_version_changed",
			)
		}
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	if targetResult.ResultVersion != stored.targetVersion+1 {
		return operations.ApprovalResult{}, operations.ErrApprovalUnavailable
	}
	updated, err := scanStoredApproval(transaction.QueryRow(ctx, `
UPDATE atlas_operations.approvals
SET status = 'executed', version = version + 1, executed_at = $3, updated_at = $3
WHERE tenant_id = $1 AND approval_id = $2
  AND status IN ('approved', 'execution_failed') AND version = $4
RETURNING `+approvalColumns,
		approval.OrganizationID.String(), approval.ApprovalID.String(), command.Now, command.ExpectedVersion,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return operations.ApprovalResult{}, errRetryApprovalTransaction
	}
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	inserted, err := transaction.Exec(ctx, `
INSERT INTO atlas_operations.approval_executions (
    execution_record_id, approval_id, tenant_id, executor_principal_id,
    executor_session_id, idempotency_key_sha256, request_sha256, outcome,
    result_approval_version, authorization_decision_id, audit_event_id,
    target_decision_id, target_audit_event_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'executed', $8, $9, $10, $11, $12, $13)
ON CONFLICT (tenant_id, executor_principal_id, idempotency_key_sha256) DO NOTHING`,
		command.ExecutionRecordID.String(), approval.ApprovalID.String(),
		approval.OrganizationID.String(), command.Actor.PrincipalID.String(),
		command.Actor.SessionID.String(), command.IdempotencyDigest[:], command.RequestDigest[:],
		updated.approval.Version, command.AuditEvent.DecisionID.String(),
		command.AuditEvent.AuditEventID.String(), targetEvent.DecisionID.String(),
		targetEvent.AuditEventID.String(), command.Now,
	)
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	if inserted.RowsAffected() != 1 {
		return operations.ApprovalResult{}, errRetryApprovalTransaction
	}
	event := command.AuditEvent
	event.ApprovalID = approval.ApprovalID
	event.SafeBeforeReference = "approval:" + string(approval.Status)
	event.SafeAfterReference = "approval:executed"
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

func validateStoredPayload(stored storedApproval) (operations.MembershipRoleChangePayload, string) {
	if stored.payloadSchemaVersion != 1 || stored.canonicalization != "rfc8785" ||
		stored.hashAlgorithm != "sha256" {
		return operations.MembershipRoleChangePayload{}, "payload_metadata_invalid"
	}
	digest := sha256.Sum256(stored.payloadCanonical)
	if subtle.ConstantTimeCompare(digest[:], stored.approval.PayloadDigest[:]) != 1 {
		return operations.MembershipRoleChangePayload{}, "payload_digest_mismatch"
	}
	payload, err := operations.DecodeCanonicalMembershipRoleChangePayload(stored.payloadCanonical)
	if err != nil {
		return operations.MembershipRoleChangePayload{}, "payload_canonicalization_invalid"
	}
	_, canonicalDigest, err := operations.CanonicalMembershipRoleChangePayload(payload)
	if err != nil || subtle.ConstantTimeCompare(canonicalDigest[:], stored.approval.PayloadDigest[:]) != 1 {
		return operations.MembershipRoleChangePayload{}, "payload_revalidation_failed"
	}
	return payload, ""
}

func loadExecutionReplay(
	ctx context.Context,
	transaction pgx.Tx,
	command operations.ExecuteApprovalCommand,
) (storedApproval, []byte, identifier.ID, error) {
	var approvalIDText, decisionIDText string
	var requestDigest []byte
	err := transaction.QueryRow(ctx, `
SELECT approval_id, request_sha256, authorization_decision_id
FROM atlas_operations.approval_executions
WHERE tenant_id = $1
  AND executor_principal_id = $2
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

func (store *Store) supersedeApproval(
	ctx context.Context,
	transaction pgx.Tx,
	stored storedApproval,
	now time.Time,
	event audit.Event,
	reason string,
) (operations.ApprovalResult, error) {
	updated, err := scanStoredApproval(transaction.QueryRow(ctx, `
UPDATE atlas_operations.approvals
SET status = 'superseded', decision_reason = $3, version = version + 1, updated_at = $4
WHERE tenant_id = $1 AND approval_id = $2
  AND status IN ('approved', 'execution_failed')
RETURNING `+approvalColumns,
		stored.approval.OrganizationID.String(), stored.approval.ApprovalID.String(), reason, now,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return operations.ApprovalResult{}, errRetryApprovalTransaction
	}
	if err != nil {
		return operations.ApprovalResult{}, approvalDatabaseError(err)
	}
	event.Decision = "denied"
	event.ReasonCode = reason
	event.SafeBeforeReference = "approval:" + string(stored.approval.Status)
	event.SafeAfterReference = "approval:superseded"
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
