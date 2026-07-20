package actor

import (
	"errors"
	"testing"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestContextRequiresExplicitValidActor(t *testing.T) {
	id, err := identifier.Parse("usr_01JAT1AS00000000000001")
	if err != nil {
		t.Fatal(err)
	}
	context, err := New(TypeWorkforce, id)
	if err != nil {
		t.Fatal(err)
	}
	if context.Type() != TypeWorkforce || context.ID() != id || context.IsZero() {
		t.Fatal("actor context changed")
	}
	if _, err := New(Type("administrator"), id); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown actor type error = %v", err)
	}
	if _, err := New(TypeCustomer, identifier.ID{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("zero actor ID error = %v", err)
	}
}
