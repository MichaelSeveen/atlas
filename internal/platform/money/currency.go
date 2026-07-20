package money

import (
	"encoding/json"

	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
)

var (
	ErrInvalidCurrency = domainerror.New(
		domainerror.MustCode("MONEY_INVALID_CURRENCY"),
		domainerror.KindInvalidArgument,
		false,
	)
	ErrUnsupportedCurrency = domainerror.New(
		domainerror.MustCode("MONEY_UNSUPPORTED_CURRENCY"),
		domainerror.KindInvalidArgument,
		false,
	)
)

// Currency is a supported ISO-like three-letter currency and its explicit
// minor-unit exponent. The exponent is catalog data, never display inference.
type Currency struct {
	code     string
	exponent uint8
}

var currencyCatalog = map[string]Currency{
	"NGN": {code: "NGN", exponent: 2},
	"USD": {code: "USD", exponent: 2},
}

// ParseCurrency validates syntax and membership in the current Atlas catalog.
func ParseCurrency(code string) (Currency, error) {
	if len(code) != 3 {
		return Currency{}, ErrInvalidCurrency
	}
	for _, character := range code {
		if character < 'A' || character > 'Z' {
			return Currency{}, ErrInvalidCurrency
		}
	}
	currency, supported := currencyCatalog[code]
	if !supported {
		return Currency{}, ErrUnsupportedCurrency
	}
	return currency, nil
}

// SupportedCurrencies returns a deterministic copy of the initial catalog.
func SupportedCurrencies() []Currency {
	return []Currency{currencyCatalog["NGN"], currencyCatalog["USD"]}
}

// Code returns the canonical three-letter currency code.
func (c Currency) Code() string {
	return c.code
}

// Exponent returns the explicit number of decimal display places.
func (c Currency) Exponent() uint8 {
	return c.exponent
}

// IsZero reports whether the currency has not been initialized.
func (c Currency) IsZero() bool {
	return c.code == ""
}

// MarshalJSON encodes a currency as its code.
func (c Currency) MarshalJSON() ([]byte, error) {
	if c.IsZero() {
		return nil, ErrInvalidCurrency
	}
	return json.Marshal(c.code)
}

// UnmarshalJSON validates currency syntax and catalog membership.
func (c *Currency) UnmarshalJSON(data []byte) error {
	var code string
	if err := json.Unmarshal(data, &code); err != nil {
		return ErrInvalidCurrency
	}
	parsed, err := ParseCurrency(code)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
