package clock

import (
	"testing"
	"time"
)

func TestFixedIsDeterministicAndUTC(t *testing.T) {
	lagos := time.FixedZone("Africa/Lagos-test", 60*60)
	input := time.Date(2026, 7, 20, 14, 30, 0, 123, lagos)
	fixed := NewFixed(input)

	first := fixed.Now()
	second := fixed.Now()
	if !first.Equal(second) {
		t.Fatalf("fixed clock changed: %s != %s", first, second)
	}
	if first.Location() != time.UTC {
		t.Fatalf("location = %s, want UTC", first.Location())
	}
	if !first.Equal(input) {
		t.Fatalf("instant changed: %s != %s", first, input)
	}
}

func TestSystemReturnsUTC(t *testing.T) {
	if got := (System{}).Now(); got.Location() != time.UTC {
		t.Fatalf("location = %s, want UTC", got.Location())
	}
}

func TestClockSkewBoundaryFixture(t *testing.T) {
	boundary := time.Date(2026, 7, 20, 13, 0, 0, 0, time.UTC)
	before := NewFixed(boundary.Add(-time.Nanosecond))
	at := NewFixed(boundary)

	validBefore := before.Now().Before(boundary)
	validAt := at.Now().Before(boundary)
	if !validBefore || validAt {
		t.Fatalf("exclusive expiry boundary changed: before=%v at=%v", validBefore, validAt)
	}
}
