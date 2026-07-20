// Package money provides bounded integer minor-unit amounts with explicit
// currency metadata. It implements no wallet, ledger, pricing, or FX behavior.
package money

import (
	"encoding/json"
	"strconv"

	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
)

const (
	// MaxMinor is the largest supported absolute minor-unit value.
	MaxMinor int64 = 1<<63 - 1
	// MinMinor deliberately excludes math.MinInt64 so every valid amount can be
	// negated safely.
	MinMinor int64 = -MaxMinor
)

var (
	ErrInvalidAmount = domainerror.New(
		domainerror.MustCode("MONEY_INVALID_AMOUNT"),
		domainerror.KindInvalidArgument,
		false,
	)
	ErrCurrencyMismatch = domainerror.New(
		domainerror.MustCode("MONEY_CURRENCY_MISMATCH"),
		domainerror.KindFailedPrecondition,
		false,
	)
	ErrOverflow = domainerror.New(
		domainerror.MustCode("MONEY_OVERFLOW"),
		domainerror.KindFailedPrecondition,
		false,
	)
	ErrSignPolicy = domainerror.New(
		domainerror.MustCode("MONEY_SIGN_POLICY_VIOLATION"),
		domainerror.KindInvalidArgument,
		false,
	)
	ErrInvalidEncoding = domainerror.New(
		domainerror.MustCode("MONEY_INVALID_ENCODING"),
		domainerror.KindInvalidArgument,
		false,
	)
)

// Amount is a signed, bounded count of minor units plus explicit currency.
// Boundary-specific constructors enforce positive or non-negative policies.
type Amount struct {
	minor    int64
	currency Currency
}

// New creates a signed amount. math.MinInt64 is excluded to keep negation and
// subtraction closed over the supported range.
func New(minor int64, currency Currency) (Amount, error) {
	if currency.IsZero() {
		return Amount{}, ErrInvalidCurrency
	}
	if minor < MinMinor || minor > MaxMinor {
		return Amount{}, ErrOverflow
	}
	return Amount{minor: minor, currency: currency}, nil
}

// NewNonNegative enforces a boundary that permits zero but not negative money.
func NewNonNegative(minor int64, currency Currency) (Amount, error) {
	if minor < 0 {
		return Amount{}, ErrSignPolicy
	}
	return New(minor, currency)
}

// NewPositive enforces a boundary that requires more than zero minor units.
func NewPositive(minor int64, currency Currency) (Amount, error) {
	if minor <= 0 {
		return Amount{}, ErrSignPolicy
	}
	return New(minor, currency)
}

// Parse constructs an amount from the canonical JSON/domain representation.
func Parse(minorUnits, currencyCode string) (Amount, error) {
	currency, err := ParseCurrency(currencyCode)
	if err != nil {
		return Amount{}, err
	}
	minor, err := parseCanonicalMinor(minorUnits)
	if err != nil {
		return Amount{}, err
	}
	return New(minor, currency)
}

func parseCanonicalMinor(value string) (int64, error) {
	if value == "" || value == "-0" || value[0] == '+' {
		return 0, ErrInvalidAmount
	}
	digits := value
	if value[0] == '-' {
		digits = value[1:]
		if digits == "" {
			return 0, ErrInvalidAmount
		}
	}
	if len(digits) > 1 && digits[0] == '0' {
		return 0, ErrInvalidAmount
	}
	for _, character := range digits {
		if character < '0' || character > '9' {
			return 0, ErrInvalidAmount
		}
	}
	minor, err := strconv.ParseInt(value, 10, 64)
	if err != nil || minor < MinMinor {
		return 0, ErrOverflow
	}
	return minor, nil
}

// MinorUnits returns the exact signed minor-unit count.
func (a Amount) MinorUnits() int64 {
	return a.minor
}

// MinorUnitsString returns the canonical decimal string representation.
func (a Amount) MinorUnitsString() string {
	return strconv.FormatInt(a.minor, 10)
}

// Currency returns the amount's explicit currency.
func (a Amount) Currency() Currency {
	return a.currency
}

// IsZero reports whether the amount is exactly zero. A zero-value Amount is
// invalid even though its numeric component is zero.
func (a Amount) IsZero() bool {
	return !a.currency.IsZero() && a.minor == 0
}

// IsNegative reports whether the amount is less than zero.
func (a Amount) IsNegative() bool {
	return a.minor < 0
}

// Add performs checked same-currency addition.
func (a Amount) Add(other Amount) (Amount, error) {
	if a.currency.IsZero() || other.currency.IsZero() {
		return Amount{}, ErrInvalidAmount
	}
	if a.currency != other.currency {
		return Amount{}, ErrCurrencyMismatch
	}
	if (other.minor > 0 && a.minor > MaxMinor-other.minor) ||
		(other.minor < 0 && a.minor < MinMinor-other.minor) {
		return Amount{}, ErrOverflow
	}
	return New(a.minor+other.minor, a.currency)
}

// Subtract performs checked same-currency subtraction.
func (a Amount) Subtract(other Amount) (Amount, error) {
	negated, err := other.Negate()
	if err != nil {
		return Amount{}, err
	}
	return a.Add(negated)
}

// Negate reverses the sign without leaving the supported range.
func (a Amount) Negate() (Amount, error) {
	if a.currency.IsZero() {
		return Amount{}, ErrInvalidAmount
	}
	return New(-a.minor, a.currency)
}

// Compare returns -1, 0, or 1 for same-currency amounts.
func (a Amount) Compare(other Amount) (int, error) {
	if a.currency.IsZero() || other.currency.IsZero() {
		return 0, ErrInvalidAmount
	}
	if a.currency != other.currency {
		return 0, ErrCurrencyMismatch
	}
	switch {
	case a.minor < other.minor:
		return -1, nil
	case a.minor > other.minor:
		return 1, nil
	default:
		return 0, nil
	}
}

type wireAmount struct {
	MinorUnits string `json:"amount_minor"`
	Currency   string `json:"currency"`
}

// MarshalJSON encodes minor units as a decimal string and currency explicitly.
func (a Amount) MarshalJSON() ([]byte, error) {
	if a.currency.IsZero() {
		return nil, ErrInvalidAmount
	}
	return json.Marshal(wireAmount{
		MinorUnits: a.MinorUnitsString(),
		Currency:   a.currency.Code(),
	})
}

// UnmarshalJSON rejects unknown/missing fields and validates canonical values.
func (a *Amount) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || len(fields) != 2 {
		return ErrInvalidEncoding
	}
	minorData, hasMinor := fields["amount_minor"]
	currencyData, hasCurrency := fields["currency"]
	if !hasMinor || !hasCurrency {
		return ErrInvalidEncoding
	}
	var minorUnits string
	var currencyCode string
	if err := json.Unmarshal(minorData, &minorUnits); err != nil {
		return ErrInvalidEncoding
	}
	if err := json.Unmarshal(currencyData, &currencyCode); err != nil {
		return ErrInvalidEncoding
	}
	parsed, err := Parse(minorUnits, currencyCode)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}
