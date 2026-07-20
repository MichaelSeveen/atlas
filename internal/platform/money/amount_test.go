package money

import (
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"testing"
)

func mustCurrency(t *testing.T, code string) Currency {
	t.Helper()
	currency, err := ParseCurrency(code)
	if err != nil {
		t.Fatalf("ParseCurrency(%q): %v", code, err)
	}
	return currency
}

func mustAmount(t *testing.T, minor int64, currency Currency) Amount {
	t.Helper()
	amount, err := New(minor, currency)
	if err != nil {
		t.Fatalf("New(%d): %v", minor, err)
	}
	return amount
}

func TestCurrencyCatalog(t *testing.T) {
	for _, test := range []struct {
		code     string
		exponent uint8
	}{
		{code: "NGN", exponent: 2},
		{code: "USD", exponent: 2},
	} {
		currency, err := ParseCurrency(test.code)
		if err != nil {
			t.Fatalf("ParseCurrency(%q): %v", test.code, err)
		}
		if currency.Exponent() != test.exponent {
			t.Errorf("%s exponent = %d, want %d", test.code, currency.Exponent(), test.exponent)
		}
	}
	if _, err := ParseCurrency("EUR"); !errors.Is(err, ErrUnsupportedCurrency) {
		t.Fatalf("unsupported error = %v", err)
	}
	if _, err := ParseCurrency("ngn"); !errors.Is(err, ErrInvalidCurrency) {
		t.Fatalf("invalid error = %v", err)
	}
}

func TestParseCanonicalMinorUnits(t *testing.T) {
	tests := []struct {
		value   string
		want    int64
		wantErr error
	}{
		{value: "0", want: 0},
		{value: "1", want: 1},
		{value: "-1", want: -1},
		{value: "9007199254740993", want: 9007199254740993},
		{value: "9223372036854775807", want: MaxMinor},
		{value: "-9223372036854775807", want: MinMinor},
		{value: "", wantErr: ErrInvalidAmount},
		{value: "+1", wantErr: ErrInvalidAmount},
		{value: "01", wantErr: ErrInvalidAmount},
		{value: "-0", wantErr: ErrInvalidAmount},
		{value: "1.0", wantErr: ErrInvalidAmount},
		{value: "1,000", wantErr: ErrInvalidAmount},
		{value: "١٠٠", wantErr: ErrInvalidAmount},
		{value: " 1", wantErr: ErrInvalidAmount},
		{value: "9223372036854775808", wantErr: ErrOverflow},
		{value: "-9223372036854775808", wantErr: ErrOverflow},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			amount, err := Parse(test.value, "NGN")
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if amount.MinorUnits() != test.want || amount.MinorUnitsString() != test.value {
				t.Fatalf("amount = %d/%q, want %d/%q", amount.MinorUnits(), amount.MinorUnitsString(), test.want, test.value)
			}
		})
	}
}

func TestCheckedArithmetic(t *testing.T) {
	ngn := mustCurrency(t, "NGN")
	usd := mustCurrency(t, "USD")

	sum, err := mustAmount(t, 7, ngn).Add(mustAmount(t, -2, ngn))
	if err != nil || sum.MinorUnits() != 5 {
		t.Fatalf("sum = %v, %v", sum, err)
	}
	difference, err := mustAmount(t, 7, ngn).Subtract(mustAmount(t, 2, ngn))
	if err != nil || difference.MinorUnits() != 5 {
		t.Fatalf("difference = %v, %v", difference, err)
	}
	if _, err := mustAmount(t, 1, ngn).Add(mustAmount(t, 1, usd)); !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("currency mismatch = %v", err)
	}
	if _, err := mustAmount(t, MaxMinor, ngn).Add(mustAmount(t, 1, ngn)); !errors.Is(err, ErrOverflow) {
		t.Fatalf("positive overflow = %v", err)
	}
	if _, err := mustAmount(t, MinMinor, ngn).Add(mustAmount(t, -1, ngn)); !errors.Is(err, ErrOverflow) {
		t.Fatalf("negative overflow = %v", err)
	}
	if negated, err := mustAmount(t, MinMinor, ngn).Negate(); err != nil || negated.MinorUnits() != MaxMinor {
		t.Fatalf("negate = %v, %v", negated, err)
	}
}

func TestSignPolicies(t *testing.T) {
	ngn := mustCurrency(t, "NGN")
	if _, err := NewNonNegative(-1, ngn); !errors.Is(err, ErrSignPolicy) {
		t.Fatalf("negative non-negative amount = %v", err)
	}
	if _, err := NewPositive(0, ngn); !errors.Is(err, ErrSignPolicy) {
		t.Fatalf("zero positive amount = %v", err)
	}
	if _, err := NewPositive(1, ngn); err != nil {
		t.Fatalf("positive amount = %v", err)
	}
}

func TestJSONContractFixtures(t *testing.T) {
	fixtureData, err := os.ReadFile("testdata/contract-fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var rawFixtures []json.RawMessage
	if err := json.Unmarshal(fixtureData, &rawFixtures); err != nil {
		t.Fatal(err)
	}
	for _, raw := range rawFixtures {
		var amount Amount
		if err := json.Unmarshal(raw, &amount); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
		roundTrip, err := json.Marshal(amount)
		if err != nil {
			t.Fatal(err)
		}
		var got, want map[string]string
		if err := json.Unmarshal(roundTrip, &got); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &want); err != nil {
			t.Fatal(err)
		}
		if got["amount_minor"] != want["amount_minor"] || got["currency"] != want["currency"] {
			t.Fatalf("round trip = %s, want %s", roundTrip, raw)
		}
	}
}

func TestJSONRejectsUnsafeShapes(t *testing.T) {
	for _, input := range []string{
		`{"amount_minor": 100, "currency": "NGN"}`,
		`{"amount_minor": "100", "currency": "NGN", "extra": "unsafe"}`,
		`{"amount_minor": "100"}`,
		`{"amount_minor": "01", "currency": "NGN"}`,
	} {
		var amount Amount
		if err := json.Unmarshal([]byte(input), &amount); err == nil {
			t.Fatalf("unsafe JSON accepted: %s", input)
		}
	}
}

func FuzzParseMinorUnits(f *testing.F) {
	for _, seed := range []string{
		"0",
		"1",
		"-1",
		"9007199254740993",
		"9223372036854775807",
		"-9223372036854775807",
		"9223372036854775808",
		"01",
		"1.0",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		amount, err := Parse(value, "NGN")
		if err != nil {
			return
		}
		if amount.MinorUnitsString() != value {
			t.Fatalf("successful parse was not canonical: %q != %q", amount.MinorUnitsString(), value)
		}
	})
}

func FuzzCheckedAddition(f *testing.F) {
	for _, seed := range [][2]int64{
		{0, 0},
		{1, -1},
		{MaxMinor, 1},
		{MinMinor, -1},
		{9007199254740993, 1},
	} {
		f.Add(seed[0], seed[1])
	}

	f.Fuzz(func(t *testing.T, left, right int64) {
		ngn := mustCurrency(t, "NGN")
		leftAmount, leftErr := New(left, ngn)
		rightAmount, rightErr := New(right, ngn)
		if leftErr != nil || rightErr != nil {
			return
		}

		want := new(big.Int).Add(big.NewInt(left), big.NewInt(right))
		max := big.NewInt(MaxMinor)
		min := big.NewInt(MinMinor)
		result, err := leftAmount.Add(rightAmount)
		outOfRange := want.Cmp(max) > 0 || want.Cmp(min) < 0
		if outOfRange {
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("%d + %d error = %v, want overflow", left, right, err)
			}
			return
		}
		if err != nil {
			t.Fatalf("%d + %d: %v", left, right, err)
		}
		if got := big.NewInt(result.MinorUnits()); got.Cmp(want) != 0 {
			t.Fatalf("%d + %d = %s, want %s", left, right, got, want)
		}

		reversed, err := rightAmount.Add(leftAmount)
		if err != nil || reversed != result {
			t.Fatalf("addition is not commutative: reverse=%v error=%v", reversed, err)
		}
	})
}
