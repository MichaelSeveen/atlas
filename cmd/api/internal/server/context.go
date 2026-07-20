package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/MichaelSeveen/atlas/internal/platform/correlation"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type requestContextKey struct{}

type requestContext struct {
	correlation correlation.Context
	trace       traceContext
}

func withRequestContext(ctx context.Context, value requestContext) context.Context {
	return context.WithValue(ctx, requestContextKey{}, value)
}

func requestContextFrom(ctx context.Context) (requestContext, bool) {
	value, ok := ctx.Value(requestContextKey{}).(requestContext)
	return value, ok
}

func acceptedOpaqueHeader(header http.Header, name, prefix string) (identifier.ID, bool) {
	values := header.Values(name)
	if len(values) != 1 {
		return identifier.ID{}, false
	}
	parsed, err := identifier.Parse(values[0])
	if err != nil || parsed.Prefix() != prefix {
		return identifier.ID{}, false
	}
	return parsed, true
}

func (a *App) requestIDs(request *http.Request) (correlation.Context, error) {
	requestID, accepted := acceptedOpaqueHeader(request.Header, "X-Request-Id", "req")
	if !accepted {
		generated, err := generatedID(a.newID, "req")
		if err != nil {
			return a.emergencyContext, err
		}
		requestID = generated
	}

	correlationID, accepted := acceptedOpaqueHeader(request.Header, "X-Correlation-Id", "cor")
	if !accepted {
		generated, err := generatedID(a.newID, "cor")
		if err != nil {
			return a.emergencyContext, err
		}
		correlationID = generated
	}

	return correlation.New(requestID, correlationID)
}

func generatedID(generator IDGenerator, prefix string) (identifier.ID, error) {
	generated, err := generator(prefix)
	if err != nil || generated.IsZero() || generated.Prefix() != prefix {
		return identifier.ID{}, errors.New("opaque identifier generation failed")
	}
	return generated, nil
}
