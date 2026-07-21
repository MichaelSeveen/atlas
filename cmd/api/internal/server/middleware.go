package server

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/MichaelSeveen/atlas/internal/platform/logging"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	metricapi "go.opentelemetry.io/otel/metric"
	traceapi "go.opentelemetry.io/otel/trace"
)

func (a *App) Handler() http.Handler {
	var handler http.Handler = http.HandlerFunc(a.route)
	handler = a.resourceLimits(handler)
	handler = a.corsMiddleware(handler)
	handler = a.recoverPanics(handler)
	handler = a.requestMetadata(handler)
	handler = securityHeaders(handler)
	return handler
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Cache-Control", "no-store")
		headers.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		headers.Set("Cross-Origin-Resource-Policy", "same-origin")
		headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(response, request)
	})
}

type responseCapture struct {
	http.ResponseWriter
	status int
}

func (r *responseCapture) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseCapture) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(body)
}

func (a *App) requestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := a.clock.Now()
		correlationContext, generationErr := a.requestIDs(request)
		route := telemetryRoute(request.URL.Path)
		method := telemetryMethod(request.Method)
		traceContext := a.fallbackTrace()
		var span traceapi.Span
		if a.tracer != nil {
			parent := extractParent(request.Context(), request.Header, a.propagator)
			spanContext, startedSpan := a.tracer.Start(parent, method+" "+route,
				traceapi.WithSpanKind(traceapi.SpanKindServer),
				traceapi.WithTimestamp(started),
				traceapi.WithAttributes(
					attribute.String("atlas.request_id", correlationContext.RequestID().String()),
					attribute.String("atlas.correlation_id", correlationContext.CorrelationID().String()),
					attribute.String("http.request.method", method),
					attribute.String("http.route", route),
				),
			)
			request = request.WithContext(spanContext)
			span = startedSpan
			if propagated := spanTraceContext(span); propagated.valid() {
				traceContext = propagated
			}
		}
		state := requestContext{correlation: correlationContext, trace: traceContext}
		request = request.WithContext(withRequestContext(request.Context(), state))

		response.Header().Set("X-Request-Id", correlationContext.RequestID().String())
		response.Header().Set("X-Correlation-Id", correlationContext.CorrelationID().String())
		if traceContext.valid() {
			response.Header().Set("traceparent", traceContext.header())
		}

		capture := &responseCapture{ResponseWriter: response}
		if generationErr != nil {
			a.writeProblem(capture, request, http.StatusServiceUnavailable, "dependency-degraded", "Service unavailable", "DEPENDENCY_DEGRADED", true)
		} else {
			next.ServeHTTP(capture, request)
		}
		if capture.status == 0 {
			capture.status = http.StatusOK
		}
		ended := a.clock.Now()
		duration := ended.Sub(started)
		if duration < 0 {
			duration = 0
		}
		outcome := statusOutcome(capture.status)
		attributes := requestAttributes(method, route, outcome, capture.status)
		safeAdd(a.requestCounter, request.Context(), 1, metricapi.WithAttributes(attributes...))
		safeRecord(a.requestDuration, request.Context(), duration.Seconds(), metricapi.WithAttributes(attributes...))
		severity := logging.SeverityInfo
		if capture.status >= 500 {
			severity = logging.SeverityError
		}
		safeLog(a.logs, logging.Record{
			Timestamp: ended, Event: logging.EventHTTPRequestCompleted, Severity: severity,
			Module: "api", Outcome: outcome,
			RequestID: correlationContext.RequestID().String(), CorrelationID: correlationContext.CorrelationID().String(),
			TraceID: traceContext.TraceID, Method: method, Route: route, StatusCode: capture.status,
			SourceRevision: a.build.SourceRevision,
		})
		if span != nil {
			span.SetAttributes(attributes...)
			if capture.status >= 500 {
				span.SetStatus(codes.Error, outcome)
			}
			span.End(traceapi.WithTimestamp(ended))
		}
	})
}

func telemetryRoute(path string) string {
	for _, route := range foundationRoutes {
		if path == route {
			return route
		}
	}
	return "unmatched"
}

func telemetryMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodOptions:
		return method
	default:
		return "OTHER"
	}
}

func statusOutcome(status int) string {
	if status >= 500 {
		return "error"
	}
	if status >= 400 {
		return "rejected"
	}
	return "ok"
}

func (a *App) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer func() {
			if recover() != nil {
				a.writeProblem(response, request, http.StatusInternalServerError, "internal-error", "Internal error", "INTERNAL_ERROR", true)
			}
		}()
		next.ServeHTTP(response, request)
	})
}

func (a *App) resourceLimits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		encoding := strings.TrimSpace(strings.ToLower(request.Header.Get("Content-Encoding")))
		if encoding != "" && encoding != "identity" {
			a.writeProblem(response, request, http.StatusUnsupportedMediaType, "unsupported-media-type", "Unsupported media type", "UNSUPPORTED_MEDIA_TYPE", false)
			return
		}
		if request.ContentLength > a.maxBodyBytes {
			a.writeProblem(response, request, http.StatusRequestEntityTooLarge, "request-too-large", "Request too large", "REQUEST_TOO_LARGE", false)
			return
		}
		if request.Body != nil && request.Body != http.NoBody {
			request.Body = http.MaxBytesReader(response, request.Body, a.maxBodyBytes)
			body, err := io.ReadAll(request.Body)
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				a.writeProblem(response, request, http.StatusRequestEntityTooLarge, "request-too-large", "Request too large", "REQUEST_TOO_LARGE", false)
				return
			}
			if err != nil || len(body) > 0 {
				a.writeProblem(response, request, http.StatusBadRequest, "request-malformed", "Malformed request", "REQUEST_MALFORMED", false)
				return
			}
		}
		next.ServeHTTP(response, request)
	})
}
