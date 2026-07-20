// Package domainerror defines stable, data-minimizing domain error vocabulary.
// Errors intentionally carry no arbitrary message, cause, or metadata.
package domainerror

import "errors"

const codeRule = "domain error code must match [A-Z0-9_]{3,120}"

// Code is a validated stable machine-readable domain error code.
type Code struct {
	value string
}

// ParseCode validates a domain error code without echoing rejected input.
func ParseCode(value string) (Code, error) {
	if len(value) < 3 || len(value) > 120 {
		return Code{}, errors.New(codeRule)
	}
	for _, character := range value {
		if (character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') &&
			character != '_' {
			return Code{}, errors.New(codeRule)
		}
	}
	return Code{value: value}, nil
}

// MustCode creates a code for package-level declarations and panics on a
// programmer-authored invalid value.
func MustCode(value string) Code {
	code, err := ParseCode(value)
	if err != nil {
		panic(err)
	}
	return code
}

// String returns the stable machine-readable value.
func (c Code) String() string {
	return c.value
}

// IsZero reports whether the code has not been initialized.
func (c Code) IsZero() bool {
	return c.value == ""
}

// Kind classifies a domain failure without coupling it to an HTTP transport.
type Kind string

const (
	KindInvalidArgument    Kind = "invalid_argument"
	KindFailedPrecondition Kind = "failed_precondition"
	KindConflict           Kind = "conflict"
	KindNotFound           Kind = "not_found"
	KindUnauthenticated    Kind = "unauthenticated"
	KindPermissionDenied   Kind = "permission_denied"
	KindRateLimited        Kind = "rate_limited"
	KindUnavailable        Kind = "unavailable"
	KindInternal           Kind = "internal"
)

// Valid reports whether the kind belongs to the closed foundation vocabulary.
func (k Kind) Valid() bool {
	switch k {
	case KindInvalidArgument,
		KindFailedPrecondition,
		KindConflict,
		KindNotFound,
		KindUnauthenticated,
		KindPermissionDenied,
		KindRateLimited,
		KindUnavailable,
		KindInternal:
		return true
	default:
		return false
	}
}

// Error is a stable, safe-to-render domain failure. It deliberately excludes
// arbitrary detail and wrapped causes so sensitive values cannot accidentally
// cross an error or telemetry boundary through this primitive.
type Error struct {
	code      Code
	kind      Kind
	retryable bool
}

// New constructs a safe domain error and panics only for invalid static
// definitions supplied by the programmer.
func New(code Code, kind Kind, retryable bool) Error {
	if code.IsZero() {
		panic(codeRule)
	}
	if !kind.Valid() {
		panic("invalid domain error kind")
	}
	return Error{code: code, kind: kind, retryable: retryable}
}

// Error returns only the stable code.
func (e Error) Error() string {
	return e.code.String()
}

// Code returns the stable machine-readable code.
func (e Error) Code() Code {
	return e.code
}

// Kind returns the transport-independent failure classification.
func (e Error) Kind() Kind {
	return e.kind
}

// Retryable reports whether retry can be considered by the owning workflow.
// It is not permission to retry a non-idempotent operation blindly.
func (e Error) Retryable() bool {
	return e.retryable
}

// Is allows errors.Is comparisons by stable code.
func (e Error) Is(target error) bool {
	switch typed := target.(type) {
	case Error:
		return e.code == typed.code
	case *Error:
		return typed != nil && e.code == typed.code
	default:
		return false
	}
}
