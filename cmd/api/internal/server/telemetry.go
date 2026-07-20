package server

import (
	"context"
	"encoding/hex"
	"io"
	"strings"
	"time"
)

type traceContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Flags        string
}

func (t traceContext) valid() bool {
	return t.TraceID != "" && t.SpanID != ""
}

func (t traceContext) header() string {
	if !t.valid() {
		return ""
	}
	return "00-" + t.TraceID + "-" + t.SpanID + "-" + t.Flags
}

type Span struct {
	Name          string
	TraceID       string
	SpanID        string
	ParentSpanID  string
	RequestID     string
	CorrelationID string
	Route         string
	Outcome       string
	StatusCode    int
	StartedAt     time.Time
	EndedAt       time.Time
}

type TraceRecorder interface {
	Record(Span)
}

type discardTraceRecorder struct{}

func (discardTraceRecorder) Record(Span) {}

type RequestObservation struct {
	Method     string
	Route      string
	StatusCode int
	Duration   time.Duration
}

type MetricsRecorder interface {
	ObserveRequest(RequestObservation)
}

type discardMetricsRecorder struct{}

func (discardMetricsRecorder) ObserveRequest(RequestObservation) {}

func safeRecord(recorder TraceRecorder, span Span) {
	defer func() { _ = recover() }()
	recorder.Record(span)
}

func safeObserve(recorder MetricsRecorder, observation RequestObservation) {
	defer func() { _ = recover() }()
	recorder.ObserveRequest(observation)
}

func parseTraceparent(value string) (traceID, parentSpanID, flags string, ok bool) {
	if len(value) != 55 || value != strings.ToLower(value) {
		return "", "", "", false
	}
	parts := strings.Split(value, "-")
	if len(parts) != 4 || parts[0] != "00" || len(parts[1]) != 32 || len(parts[2]) != 16 || len(parts[3]) != 2 {
		return "", "", "", false
	}
	for _, part := range parts {
		if _, err := hex.DecodeString(part); err != nil {
			return "", "", "", false
		}
	}
	if allZero(parts[1]) || allZero(parts[2]) {
		return "", "", "", false
	}
	return parts[1], parts[2], parts[3], true
}

func allZero(value string) bool {
	for _, character := range value {
		if character != '0' {
			return false
		}
	}
	return true
}

func (a *App) randomHex(bytes int) (string, error) {
	random := make([]byte, bytes)
	a.entropyMu.Lock()
	_, err := io.ReadFull(a.entropy, random)
	a.entropyMu.Unlock()
	if err != nil {
		return "", err
	}
	if allZero(hex.EncodeToString(random)) {
		random[len(random)-1] = 1
	}
	return hex.EncodeToString(random), nil
}

func (a *App) startTrace(incoming string) traceContext {
	traceID, parentSpanID, flags, accepted := parseTraceparent(incoming)
	if !accepted {
		generated, err := a.randomHex(16)
		if err != nil {
			return traceContext{}
		}
		traceID = generated
		parentSpanID = ""
		flags = "00"
	}
	spanID, err := a.randomHex(8)
	if err != nil {
		return traceContext{}
	}
	return traceContext{TraceID: traceID, SpanID: spanID, ParentSpanID: parentSpanID, Flags: flags}
}

func (a *App) readinessState(ctx context.Context) ReadinessState {
	started := a.clock.Now()
	checkContext, cancel := context.WithTimeout(ctx, a.readinessTimeout)
	defer cancel()
	state := a.readiness.Check(checkContext)
	ended := a.clock.Now()

	requestState, found := requestContextFrom(ctx)
	if !found || !requestState.trace.valid() {
		return state
	}
	childSpanID, err := a.randomHex(8)
	if err != nil {
		return state
	}
	outcome := "not_ready"
	if state.Ready() {
		outcome = "ready"
	}
	safeRecord(a.traces, Span{
		Name:          "readiness.check",
		TraceID:       requestState.trace.TraceID,
		SpanID:        childSpanID,
		ParentSpanID:  requestState.trace.SpanID,
		RequestID:     requestState.correlation.RequestID().String(),
		CorrelationID: requestState.correlation.CorrelationID().String(),
		Route:         "/health/ready",
		Outcome:       outcome,
		StartedAt:     started,
		EndedAt:       ended,
	})
	return state
}
