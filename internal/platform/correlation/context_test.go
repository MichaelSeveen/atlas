package correlation

import (
	"errors"
	"testing"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func mustID(t *testing.T, value string) identifier.ID {
	t.Helper()
	id, err := identifier.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestContextIsExplicitImmutableAndSafe(t *testing.T) {
	requestID := mustID(t, "req_01JAT1AS00000000000001")
	correlationID := mustID(t, "cor_01JAT1AS00000000000002")
	causationID := mustID(t, "evt_01JAT1AS00000000000003")

	original, err := New(requestID, correlationID)
	if err != nil {
		t.Fatal(err)
	}
	withCause, err := original.WithCausation(causationID)
	if err != nil {
		t.Fatal(err)
	}
	if !original.CausationID().IsZero() {
		t.Fatal("WithCausation mutated the original context")
	}
	fields := withCause.SafeFields()
	if fields.RequestID != requestID.String() ||
		fields.CorrelationID != correlationID.String() ||
		fields.CausationID != causationID.String() {
		t.Fatalf("safe fields changed: %+v", fields)
	}
}

func TestContextFailsClosed(t *testing.T) {
	valid := mustID(t, "req_01JAT1AS00000000000001")
	if _, err := New(identifier.ID{}, valid); !errors.Is(err, ErrInvalid) {
		t.Fatalf("zero request error = %v", err)
	}
	if _, err := New(valid, identifier.ID{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("zero correlation error = %v", err)
	}
}
