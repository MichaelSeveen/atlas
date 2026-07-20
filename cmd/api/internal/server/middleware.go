package server

import (
	"errors"
	"io"
	"net/http"
	"strings"
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
		traceHeader := ""
		if values := request.Header.Values("traceparent"); len(values) == 1 {
			traceHeader = values[0]
		}
		trace := a.startTrace(traceHeader)
		state := requestContext{correlation: correlationContext, trace: trace}
		request = request.WithContext(withRequestContext(request.Context(), state))

		response.Header().Set("X-Request-Id", correlationContext.RequestID().String())
		response.Header().Set("X-Correlation-Id", correlationContext.CorrelationID().String())
		if trace.valid() {
			response.Header().Set("traceparent", trace.header())
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
		route := telemetryRoute(request.URL.Path)
		duration := ended.Sub(started)
		if duration < 0 {
			duration = 0
		}
		safeObserve(a.metrics, RequestObservation{
			Method:     request.Method,
			Route:      route,
			StatusCode: capture.status,
			Duration:   duration,
		})
		if trace.valid() {
			safeRecord(a.traces, Span{
				Name:          request.Method + " " + route,
				TraceID:       trace.TraceID,
				SpanID:        trace.SpanID,
				ParentSpanID:  trace.ParentSpanID,
				RequestID:     correlationContext.RequestID().String(),
				CorrelationID: correlationContext.CorrelationID().String(),
				Route:         route,
				Outcome:       statusOutcome(capture.status),
				StatusCode:    capture.status,
				StartedAt:     started,
				EndedAt:       ended,
			})
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
