// Package identifier provides opaque, contract-compatible identifiers.
package identifier

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"io"
	"strings"

	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
)

const (
	minimumPrefixLength = 2
	maximumPrefixLength = 8
	minimumBodyLength   = 20
	maximumBodyLength   = 32
	generatedBytes      = 16
)

var (
	crockfordEncoding = base32.NewEncoding("0123456789ABCDEFGHJKMNPQRSTVWXYZ").WithPadding(base32.NoPadding)

	ErrInvalid = domainerror.New(
		domainerror.MustCode("IDENTIFIER_INVALID"),
		domainerror.KindInvalidArgument,
		false,
	)
	ErrGenerationFailed = domainerror.New(
		domainerror.MustCode("IDENTIFIER_GENERATION_FAILED"),
		domainerror.KindUnavailable,
		true,
	)
)

// ID is an opaque identifier matching the canonical contract shape:
// lowercase prefix, underscore, and Crockford Base32 body.
type ID struct {
	value string
}

// New creates a 128-bit cryptographically random opaque identifier.
func New(prefix string) (ID, error) {
	return newWithReader(prefix, rand.Reader)
}

func newWithReader(prefix string, reader io.Reader) (ID, error) {
	if !validPrefix(prefix) {
		return ID{}, ErrInvalid
	}
	random := make([]byte, generatedBytes)
	if _, err := io.ReadFull(reader, random); err != nil {
		return ID{}, ErrGenerationFailed
	}
	return Parse(prefix + "_" + crockfordEncoding.EncodeToString(random))
}

// Parse validates an opaque identifier against the canonical contract pattern.
func Parse(value string) (ID, error) {
	if strings.Count(value, "_") != 1 {
		return ID{}, ErrInvalid
	}
	prefix, body, found := strings.Cut(value, "_")
	if !found || !validPrefix(prefix) || !validBody(body) {
		return ID{}, ErrInvalid
	}
	return ID{value: value}, nil
}

func validPrefix(prefix string) bool {
	if len(prefix) < minimumPrefixLength || len(prefix) > maximumPrefixLength {
		return false
	}
	for _, character := range prefix {
		if character < 'a' || character > 'z' {
			return false
		}
	}
	return true
}

func validBody(body string) bool {
	if len(body) < minimumBodyLength || len(body) > maximumBodyLength {
		return false
	}
	for _, character := range body {
		if !strings.ContainsRune("0123456789ABCDEFGHJKMNPQRSTVWXYZ", character) {
			return false
		}
	}
	return true
}

// String returns the canonical representation or an empty string for zero ID.
func (id ID) String() string {
	return id.value
}

// Prefix returns the non-semantic resource prefix.
func (id ID) Prefix() string {
	prefix, _, _ := strings.Cut(id.value, "_")
	return prefix
}

// IsZero reports whether the ID has not been initialized.
func (id ID) IsZero() bool {
	return id.value == ""
}

// MarshalText encodes the canonical identifier.
func (id ID) MarshalText() ([]byte, error) {
	if id.IsZero() {
		return nil, ErrInvalid
	}
	return []byte(id.value), nil
}

// UnmarshalText validates a canonical identifier.
func (id *ID) UnmarshalText(text []byte) error {
	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// MarshalJSON encodes IDs as JSON strings.
func (id ID) MarshalJSON() ([]byte, error) {
	if id.IsZero() {
		return nil, ErrInvalid
	}
	return json.Marshal(id.value)
}

// UnmarshalJSON rejects non-string and malformed identifier values.
func (id *ID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return ErrInvalid
	}
	return id.UnmarshalText([]byte(value))
}
