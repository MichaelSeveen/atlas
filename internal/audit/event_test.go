package audit

import (
	"testing"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestAuditEventRequiresExactlyOneScopeAndUTC(t *testing.T) {
	event := validEvent(t)
	if err := event.Validate(); err != nil {
		t.Fatal(err)
	}

	event.GlobalScope = "platform-security"
	if err := event.Validate(); err == nil {
		t.Fatal("audit event with both tenant and global scope was accepted")
	}
	event = validEvent(t)
	event.TenantID = identifier.ID{}
	if err := event.Validate(); err == nil {
		t.Fatal("audit event without tenant or global scope was accepted")
	}
	event = validEvent(t)
	event.OccurredAt = event.OccurredAt.In(time.FixedZone("offset", 3600))
	if err := event.Validate(); err == nil {
		t.Fatal("non-UTC audit occurrence time was accepted")
	}
}

func validEvent(t *testing.T) Event {
	t.Helper()
	parse := func(value string) identifier.ID {
		id, err := identifier.Parse(value)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	return Event{
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
