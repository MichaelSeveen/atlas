package server

import (
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/MichaelSeveen/atlas/internal/platform/logging"
	"go.opentelemetry.io/otel/attribute"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	traceapi "go.opentelemetry.io/otel/trace"
)

type traceContext struct {
	TraceID string
	SpanID  string
	Flags   string
}

func (trace traceContext) valid() bool {
	return trace.TraceID != "" && trace.SpanID != ""
}

func (trace traceContext) header() string {
	if !trace.valid() {
		return ""
	}
	return "00-" + trace.TraceID + "-" + trace.SpanID + "-" + trace.Flags
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

func (app *App) randomHex(bytes int) (string, error) {
	random := make([]byte, bytes)
	app.entropyMu.Lock()
	_, err := io.ReadFull(app.entropy, random)
	app.entropyMu.Unlock()
	if err != nil {
		return "", err
	}
	if allZero(hex.EncodeToString(random)) {
		random[len(random)-1] = 1
	}
	return hex.EncodeToString(random), nil
}

func (app *App) fallbackTrace() traceContext {
	traceID, err := app.randomHex(16)
	if err != nil {
		return traceContext{}
	}
	spanID, err := app.randomHex(8)
	if err != nil {
		return traceContext{}
	}
	return traceContext{TraceID: traceID, SpanID: spanID, Flags: "00"}
}

func extractParent(ctx context.Context, headers http.Header, propagator propagation.TextMapPropagator) context.Context {
	values := headers.Values("traceparent")
	if len(values) != 1 {
		return ctx
	}
	if _, _, _, valid := parseTraceparent(values[0]); !valid {
		return ctx
	}
	return propagator.Extract(ctx, propagation.MapCarrier{"traceparent": values[0]})
}

func spanTraceContext(span traceapi.Span) traceContext {
	if span == nil {
		return traceContext{}
	}
	spanContext := span.SpanContext()
	if !spanContext.IsValid() {
		return traceContext{}
	}
	flags := "00"
	if spanContext.IsSampled() {
		flags = "01"
	}
	return traceContext{TraceID: spanContext.TraceID().String(), SpanID: spanContext.SpanID().String(), Flags: flags}
}

func safeAdd(counter metricapi.Int64Counter, ctx context.Context, value int64, options ...metricapi.AddOption) {
	if counter == nil {
		return
	}
	defer func() { _ = recover() }()
	counter.Add(ctx, value, options...)
}

func safeRecord(histogram metricapi.Float64Histogram, ctx context.Context, value float64, options ...metricapi.RecordOption) {
	if histogram == nil {
		return
	}
	defer func() { _ = recover() }()
	histogram.Record(ctx, value, options...)
}

func safeLog(recorder logging.Recorder, record logging.Record) {
	defer func() { _ = recover() }()
	_ = recorder.Record(record)
}

func requestAttributes(method, route, outcome string, status int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
		attribute.String("atlas.outcome", outcome),
		attribute.Int("http.response.status_code", status),
	}
}

func (app *App) readinessState(ctx context.Context) ReadinessState {
	checkContext, cancel := context.WithTimeout(ctx, app.readinessTimeout)
	defer cancel()
	var span traceapi.Span
	if app.tracer != nil {
		checkContext, span = app.tracer.Start(checkContext, "readiness.check")
		defer span.End()
	}
	state := app.readiness.Check(checkContext)
	outcome := "not_ready"
	if state.Ready() {
		outcome = "ready"
	}
	if span != nil {
		span.SetAttributes(attribute.String("atlas.outcome", outcome), attribute.String("http.route", "/health/ready"))
	}
	return state
}
