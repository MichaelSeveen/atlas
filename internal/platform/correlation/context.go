// Package correlation defines explicitly propagated request, correlation, and
// causation identifiers with no arbitrary high-cardinality metadata.
package correlation

import (
	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

var ErrInvalid = domainerror.New(
	domainerror.MustCode("CORRELATION_CONTEXT_INVALID"),
	domainerror.KindInvalidArgument,
	false,
)

// Context is immutable correlation data passed across synchronous/asynchronous
// boundaries. Causation may be absent at an originating boundary.
type Context struct {
	requestID     identifier.ID
	correlationID identifier.ID
	causationID   identifier.ID
}

// New validates required request and correlation identifiers.
func New(requestID, correlationID identifier.ID) (Context, error) {
	if requestID.IsZero() || correlationID.IsZero() {
		return Context{}, ErrInvalid
	}
	return Context{requestID: requestID, correlationID: correlationID}, nil
}

// WithCausation returns a new context with an explicit causation identifier.
func (c Context) WithCausation(causationID identifier.ID) (Context, error) {
	if c.IsZero() || causationID.IsZero() {
		return Context{}, ErrInvalid
	}
	c.causationID = causationID
	return c, nil
}

// RequestID returns the per-request identifier.
func (c Context) RequestID() identifier.ID {
	return c.requestID
}

// CorrelationID returns the cross-boundary correlation identifier.
func (c Context) CorrelationID() identifier.ID {
	return c.correlationID
}

// CausationID returns an optional causal predecessor identifier.
func (c Context) CausationID() identifier.ID {
	return c.causationID
}

// IsZero reports whether required identifiers are absent.
func (c Context) IsZero() bool {
	return c.requestID.IsZero() || c.correlationID.IsZero()
}

// Fields is a bounded safe telemetry projection containing only validated
// opaque identifiers.
type Fields struct {
	RequestID     string
	CorrelationID string
	CausationID   string
}

// SafeFields returns bounded identifier fields without user-provided labels.
func (c Context) SafeFields() Fields {
	return Fields{
		RequestID:     c.requestID.String(),
		CorrelationID: c.correlationID.String(),
		CausationID:   c.causationID.String(),
	}
}
