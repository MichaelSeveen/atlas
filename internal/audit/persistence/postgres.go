// Package persistence contains Audit-owned PostgreSQL adapters.
package persistence

import (
	"context"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
)

const insertEventSQL = `
INSERT INTO atlas_audit.audit_events (
    audit_event_id, actor_id, actor_type, tenant_id, global_scope, session_assurance,
    action, target_type, target_id, decision_id, decision, reason_code, correlation_id,
    approval_id, occurred_at, safe_before_reference, safe_after_reference
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12, $13,
    $14, $15, $16, $17
)`

var ErrWriteFailed = domainerror.New(
	domainerror.MustCode("AUDIT_WRITE_FAILED"),
	domainerror.KindUnavailable,
	true,
)

// Recorder writes audit facts through the caller's transaction.
type Recorder struct{}

// Record inserts one immutable fact and never starts or commits a transaction.
func (Recorder) Record(ctx context.Context, transaction audit.Transaction, event audit.Event) error {
	if transaction == nil {
		return ErrWriteFailed
	}
	if err := event.Validate(); err != nil {
		return err
	}
	var tenantID any
	if !event.TenantID.IsZero() {
		tenantID = event.TenantID.String()
	}
	var globalScope any
	if event.GlobalScope != "" {
		globalScope = event.GlobalScope
	}
	var approvalID any
	if !event.ApprovalID.IsZero() {
		approvalID = event.ApprovalID.String()
	}
	if _, err := transaction.Exec(
		ctx,
		insertEventSQL,
		event.AuditEventID.String(),
		event.ActorID.String(),
		event.ActorType,
		tenantID,
		globalScope,
		event.SessionAssurance,
		event.Action,
		event.TargetType,
		event.TargetID,
		event.DecisionID.String(),
		event.Decision,
		event.ReasonCode,
		event.CorrelationID.String(),
		approvalID,
		event.OccurredAt,
		nullableReference(event.SafeBeforeReference),
		nullableReference(event.SafeAfterReference),
	); err != nil {
		return ErrWriteFailed
	}
	return nil
}

func nullableReference(value string) any {
	if value == "" {
		return nil
	}
	return value
}

var _ audit.Recorder = Recorder{}
