// Package audit owns append-only Atlas security and access facts.
package audit

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

var ErrEventInvalid = domainerror.New(
	domainerror.MustCode("AUDIT_EVENT_INVALID"),
	domainerror.KindInvalidArgument,
	false,
)

// Transaction is the explicit caller-owned PostgreSQL transaction required for atomic audit writes.
type Transaction interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// Recorder is the synchronous application boundary used by privileged mutations.
type Recorder interface {
	Record(context.Context, Transaction, Event) error
}

// Event contains the minimum safe, append-only fields ratified by ADR 0014.
type Event struct {
	AuditEventID        identifier.ID
	ActorID             identifier.ID
	ActorType           string
	TenantID            identifier.ID
	GlobalScope         string
	SessionAssurance    string
	Action              string
	TargetType          string
	TargetID            string
	DecisionID          identifier.ID
	Decision            string
	ReasonCode          string
	CorrelationID       identifier.ID
	ApprovalID          identifier.ID
	OccurredAt          time.Time
	SafeBeforeReference string
	SafeAfterReference  string
}

// Validate rejects incomplete, ambiguous-scope, or non-UTC audit facts before persistence.
func (event Event) Validate() error {
	if event.AuditEventID.IsZero() || event.AuditEventID.Prefix() != "aud" ||
		event.ActorID.IsZero() ||
		event.DecisionID.IsZero() || event.DecisionID.Prefix() != "dec" ||
		event.CorrelationID.IsZero() || event.CorrelationID.Prefix() != "cor" ||
		event.OccurredAt.IsZero() || event.OccurredAt.Location() != time.UTC ||
		strings.TrimSpace(event.Action) == "" ||
		strings.TrimSpace(event.TargetType) == "" ||
		strings.TrimSpace(event.TargetID) == "" ||
		strings.TrimSpace(event.ReasonCode) == "" {
		return ErrEventInvalid
	}
	if event.TenantID.IsZero() == (strings.TrimSpace(event.GlobalScope) == "") {
		return ErrEventInvalid
	}
	switch event.ActorType {
	case "customer", "merchant", "workforce", "machine", "system":
	default:
		return ErrEventInvalid
	}
	switch event.SessionAssurance {
	case "none", "baseline", "stepped-up", "phishing-resistant":
	default:
		return ErrEventInvalid
	}
	switch event.Decision {
	case "allowed", "denied", "executed":
	default:
		return ErrEventInvalid
	}
	return nil
}
