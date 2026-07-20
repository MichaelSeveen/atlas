package domainerror

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCode(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid", value: "MONEY_OVERFLOW"},
		{name: "minimum length", value: "ABC"},
		{name: "lowercase", value: "money_overflow", wantErr: true},
		{name: "punctuation", value: "MONEY-OVERFLOW", wantErr: true},
		{name: "too short", value: "AB", wantErr: true},
		{name: "too long", value: strings.Repeat("A", 121), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, err := ParseCode(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseCode() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && code.String() != test.value {
				t.Fatalf("ParseCode() = %q, want %q", code.String(), test.value)
			}
		})
	}
}

func TestErrorIsStableAndDataMinimizing(t *testing.T) {
	code := MustCode("CUSTOMER_LOOKUP_FAILED")
	failure := New(code, KindInternal, false)

	if got := failure.Error(); got != "CUSTOMER_LOOKUP_FAILED" {
		t.Fatalf("Error() = %q", got)
	}
	if strings.Contains(failure.Error(), "person@example.test") {
		t.Fatal("domain error rendered sensitive data")
	}
	if failure.Kind() != KindInternal || failure.Retryable() {
		t.Fatal("domain error classification changed")
	}
	if !errors.Is(failure, New(code, KindUnavailable, true)) {
		t.Fatal("errors.Is must compare stable codes")
	}
}

func TestValidationErrorDoesNotEchoRejectedInput(t *testing.T) {
	sensitive := "person@example.test"
	_, err := ParseCode(sensitive)
	if err == nil {
		t.Fatal("invalid code accepted")
	}
	if strings.Contains(err.Error(), sensitive) || strings.Contains(err.Error(), "person@example") {
		t.Fatalf("validation error echoed rejected input: %q", err)
	}
}

func TestKindVocabularyIsClosed(t *testing.T) {
	valid := []Kind{
		KindInvalidArgument,
		KindFailedPrecondition,
		KindConflict,
		KindNotFound,
		KindUnauthenticated,
		KindPermissionDenied,
		KindRateLimited,
		KindUnavailable,
		KindInternal,
	}
	for _, kind := range valid {
		if !kind.Valid() {
			t.Errorf("kind %q must be valid", kind)
		}
	}
	if Kind("invented").Valid() {
		t.Fatal("unknown kind must fail closed")
	}
}

func TestMustCodeRejectsInvalidDefinition(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustCode must panic for an invalid static definition")
		}
	}()
	MustCode("not-valid")
}
