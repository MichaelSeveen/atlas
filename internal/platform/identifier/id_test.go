package identifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "minimum canonical", value: "id_00000000000000000000"},
		{name: "contract compatible", value: "wal_01JAT1AS00000000000001"},
		{name: "maximum prefix and body", value: "resource_" + strings.Repeat("Z", 32)},
		{name: "empty", value: "", wantErr: true},
		{name: "short prefix", value: "x_00000000000000000000", wantErr: true},
		{name: "long prefix", value: "resources_00000000000000000000", wantErr: true},
		{name: "uppercase prefix", value: "WAL_00000000000000000000", wantErr: true},
		{name: "short body", value: "wal_0000000000000000000", wantErr: true},
		{name: "ambiguous crockford letter", value: "wal_01JATLAS00000000000001", wantErr: true},
		{name: "lowercase body", value: "wal_01jat1as00000000000001", wantErr: true},
		{name: "multiple separators", value: "wal_extra_00000000000000000000", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id, err := Parse(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && id.String() != test.value {
				t.Fatalf("Parse() = %q, want %q", id.String(), test.value)
			}
		})
	}
}

func TestNewUsesCryptographicEntropyShape(t *testing.T) {
	first, err := newWithReader("req", bytes.NewReader(make([]byte, generatedBytes)))
	if err != nil {
		t.Fatalf("newWithReader(): %v", err)
	}
	secondBytes := bytes.Repeat([]byte{0xff}, generatedBytes)
	second, err := newWithReader("req", bytes.NewReader(secondBytes))
	if err != nil {
		t.Fatalf("newWithReader(): %v", err)
	}
	if first == second {
		t.Fatal("different 128-bit entropy produced the same identifier")
	}
	if len(strings.TrimPrefix(first.String(), "req_")) != 26 {
		t.Fatalf("generated body length = %d", len(first.String())-4)
	}
	if _, err := Parse(first.String()); err != nil {
		t.Fatalf("generated ID does not parse: %v", err)
	}
}

func TestNewFailsClosed(t *testing.T) {
	_, err := newWithReader("req", bytes.NewReader(nil))
	if !errors.Is(err, ErrGenerationFailed) {
		t.Fatalf("error = %v, want %v", err, ErrGenerationFailed)
	}
}

func TestParseErrorDoesNotEchoRejectedInput(t *testing.T) {
	sensitive := "person@example.test"
	_, err := Parse(sensitive)
	if err == nil {
		t.Fatal("invalid identifier accepted")
	}
	if strings.Contains(err.Error(), sensitive) || strings.Contains(err.Error(), "person@example") {
		t.Fatalf("identifier error echoed rejected input: %q", err)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	want, err := Parse("req_01JAT1AS00000000000001")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got ID
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("round trip = %q, want %q", got, want)
	}
	if err := json.Unmarshal([]byte(`123`), &got); !errors.Is(err, ErrInvalid) {
		t.Fatalf("non-string error = %v, want %v", err, ErrInvalid)
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"req_01JAT1AS00000000000001",
		"wal_00000000000000000000",
		"wal_01JATLAS00000000000001",
		"",
		"person@example.test",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		id, err := Parse(value)
		if err != nil {
			return
		}
		if id.String() != value {
			t.Fatalf("successful parse changed value: %q != %q", id.String(), value)
		}
		if _, err := Parse(id.String()); err != nil {
			t.Fatalf("successful parse did not round trip: %v", err)
		}
	})
}
