package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type captureTransaction struct {
	query string
	args  []any
	err   error
}

func (transaction *captureTransaction) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	transaction.query = query
	transaction.args = args
	return pgconn.CommandTag{}, transaction.err
}

func TestRecorderUsesCallerTransactionAndCompleteFieldSet(t *testing.T) {
	transaction := &captureTransaction{}
	event := validPersistenceEvent(t)
	if err := (Recorder{}).Record(context.Background(), transaction, event); err != nil {
		t.Fatal(err)
	}
	if transaction.query != insertEventSQL || len(transaction.args) != 17 {
		t.Fatal("audit recorder did not use the closed insert statement")
	}
}

func TestRecorderFailsClosedOnPersistenceError(t *testing.T) {
	transaction := &captureTransaction{err: errors.New("synthetic persistence outage")}
	if err := (Recorder{}).Record(context.Background(), transaction, validPersistenceEvent(t)); !errors.Is(err, ErrWriteFailed) {
		t.Fatalf("persistence failure = %v, want safe audit write failure", err)
	}
}

func validPersistenceEvent(t *testing.T) audit.Event {
	t.Helper()
	parse := func(value string) identifier.ID {
		id, err := identifier.Parse(value)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	return audit.Event{
		AuditEventID:     parse("aud_01JAT1AS00000000000002"),
		ActorID:          parse("usr_01JAT1AS00000000000003"),
		ActorType:        "workforce",
		TenantID:         parse("ten_01JAT1AS00000000000002"),
		SessionAssurance: "phishing-resistant",
		Action:           "identity.session.revoke",
		TargetType:       "session",
		TargetID:         "ses_01JAT1AS00000000000901",
		DecisionID:       parse("dec_01JAT1AS00000000000002"),
		Decision:         "executed",
		ReasonCode:       "security_review",
		CorrelationID:    parse("cor_01JAT1AS00000000000002"),
		OccurredAt:       time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
	}
}
