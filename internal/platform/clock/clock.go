// Package clock provides explicit UTC time sources for domain code.
package clock

import "time"

// Clock is the only time source domain code should depend on.
type Clock interface {
	Now() time.Time
}

// System is the production adapter around the host clock.
type System struct{}

// Now returns the current instant normalized to UTC.
func (System) Now() time.Time {
	return time.Now().UTC()
}

// Fixed is an immutable deterministic clock for tests, fixtures, and replay.
type Fixed struct {
	now time.Time
}

// NewFixed normalizes the supplied instant to UTC.
func NewFixed(now time.Time) Fixed {
	return Fixed{now: now.UTC()}
}

// Now returns the configured UTC instant.
func (f Fixed) Now() time.Time {
	return f.now
}
