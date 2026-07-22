package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/clock"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
	"github.com/MichaelSeveen/atlas/internal/platform/logging"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	metricdata "go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const (
	testSourceRevision = "710bca0a3c5dd44fb009512e2200a65b5da59dcd"
	validRequestID     = "req_01JAT1AS00000000000001"
	validCorrelationID = "cor_01JAT1AS00000000000001"
	validTraceparent   = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
)

var testBuildTime = time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

type sequentialReader struct {
	next byte
}

func (r *sequentialReader) Read(target []byte) (int, error) {
	for index := range target {
		if r.next == 0 {
			r.next = 1
		}
		target[index] = r.next
		r.next++
	}
	return len(target), nil
}

type panickingLogger struct{}

func (panickingLogger) Record(logging.Record) error { panic("person@example.test") }

func staticID(prefix string) (identifier.ID, error) {
	return identifier.Parse(prefix + "_00000000000000000001")
}

func newTestApp(t *testing.T, state ReadinessState, modify func(*Options)) *App {
	t.Helper()
	options := Options{
		Build: BuildInfo{
			SourceRevision:  testSourceRevision,
			ContractVersion: ContractVersion,
			BuildTime:       testBuildTime,
		},
		Readiness: ReadinessFunc(func(context.Context) ReadinessState { return state }),
		Clock:     clock.NewFixed(testBuildTime),
		NewID:     staticID,
		Entropy:   &sequentialReader{next: 1},
		Tracer: sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		).Tracer("atlas-server-test"),
	}
	if modify != nil {
		modify(&options)
	}
	app, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func perform(handler http.Handler, method, target string, body io.Reader) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, body)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestFoundationEndpointContract(t *testing.T) {
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, nil)
	tests := []struct {
		path       string
		wantStatus int
		wantBody   map[string]any
	}{
		{path: "/health/live", wantStatus: http.StatusOK, wantBody: map[string]any{"status": "alive"}},
		{path: "/health/ready", wantStatus: http.StatusOK, wantBody: map[string]any{"status": "ready"}},
		{path: "/version", wantStatus: http.StatusOK, wantBody: map[string]any{
			"source_revision": testSourceRevision, "contract_version": ContractVersion, "build_time": "2026-07-20T00:00:00Z",
		}},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			response := perform(app.Handler(), http.MethodGet, test.path, nil)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, body=%s", response.Code, response.Body)
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, test.wantBody) {
				t.Fatalf("body = %#v, want %#v", body, test.wantBody)
			}
			assertFoundationHeaders(t, response.Header())
		})
	}
}

func TestLiveServerSmokeHealthyAndMigrationBehind(t *testing.T) {
	tests := []struct {
		name      string
		state     ReadinessState
		readyCode int
	}{
		{
			name:      "healthy",
			state:     ReadinessState{DependenciesReady: true, MigrationsCurrent: true},
			readyCode: http.StatusOK,
		},
		{
			name:      "migration behind",
			state:     ReadinessState{DependenciesReady: true, MigrationsCurrent: false},
			readyCode: http.StatusServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			liveServer := httptest.NewServer(newTestApp(t, test.state, nil).Handler())
			defer liveServer.Close()
			client := &http.Client{Timeout: 2 * time.Second}
			for path, wantStatus := range map[string]int{
				"/health/live":  http.StatusOK,
				"/health/ready": test.readyCode,
				"/version":      http.StatusOK,
			} {
				response, err := client.Get(liveServer.URL + path)
				if err != nil {
					t.Fatalf("GET %s: %v", path, err)
				}
				_ = response.Body.Close()
				if response.StatusCode != wantStatus {
					t.Fatalf("GET %s status = %d, want %d", path, response.StatusCode, wantStatus)
				}
			}
		})
	}
}

func assertFoundationHeaders(t *testing.T, headers http.Header) {
	t.Helper()
	for name, want := range map[string]string{
		"Cache-Control":                "no-store",
		"Content-Security-Policy":      "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           "camera=(), microphone=(), geolocation=(), payment=()",
		"Referrer-Policy":              "no-referrer",
		"Strict-Transport-Security":    "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"X-Request-Id":                 "req_00000000000000000001",
		"X-Correlation-Id":             "cor_00000000000000000001",
	} {
		if got := headers.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if _, _, _, ok := parseTraceparent(headers.Get("traceparent")); !ok {
		t.Errorf("invalid response traceparent: %q", headers.Get("traceparent"))
	}
}

func TestMigrationLagFailsReadinessOnly(t *testing.T) {
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: false}, nil)
	ready := perform(app.Handler(), http.MethodGet, "/health/ready", nil)
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d", ready.Code)
	}
	assertProblem(t, ready, "DEPENDENCY_DEGRADED")
	for _, forbidden := range []string{"database", "schema", "migration", "postgres", "host"} {
		if strings.Contains(strings.ToLower(ready.Body.String()), forbidden) {
			t.Errorf("readiness leaked %q: %s", forbidden, ready.Body)
		}
	}
	if live := perform(app.Handler(), http.MethodGet, "/health/live", nil); live.Code != http.StatusOK {
		t.Fatalf("liveness status = %d", live.Code)
	}
	if version := perform(app.Handler(), http.MethodGet, "/version", nil); version.Code != http.StatusOK {
		t.Fatalf("version status = %d", version.Code)
	}
}

func TestGoldenSyntheticTraceAndBoundedMetrics(t *testing.T) {
	spans := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spans),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Tracer = tracerProvider.Tracer("atlas-golden-test")
		options.Meter = meterProvider.Meter("atlas-golden-test")
	})
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	request.Header.Set("X-Request-Id", validRequestID)
	request.Header.Set("X-Correlation-Id", validCorrelationID)
	request.Header.Set("traceparent", validTraceparent)
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Header().Get("X-Request-Id") != validRequestID || response.Header().Get("X-Correlation-Id") != validCorrelationID {
		t.Fatal("validated request metadata was not propagated")
	}
	traceID, _, _, ok := parseTraceparent(response.Header().Get("traceparent"))
	if !ok || traceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace continuity failed: %q", response.Header().Get("traceparent"))
	}

	endedSpans := spans.Ended()
	if len(endedSpans) != 2 {
		t.Fatalf("spans = %d, want readiness child + server", len(endedSpans))
	}
	var child, serverSpan sdktrace.ReadOnlySpan
	for _, span := range endedSpans {
		switch span.Name() {
		case "readiness.check":
			child = span
		case "GET /health/ready":
			serverSpan = span
		}
	}
	if child == nil || serverSpan == nil {
		t.Fatal("golden span names were not exported")
	}
	if child.SpanContext().TraceID() != serverSpan.SpanContext().TraceID() || child.Parent().SpanID() != serverSpan.SpanContext().SpanID() || serverSpan.Parent().SpanID().String() != "00f067aa0ba902b7" {
		t.Fatalf("trace linkage changed: child=%s/%s server=%s/%s", child.SpanContext().TraceID(), child.Parent().SpanID(), serverSpan.SpanContext().SpanID(), serverSpan.Parent().SpanID())
	}
	if child.SpanContext().SpanID() == serverSpan.SpanContext().SpanID() {
		t.Fatal("golden trace reused a span identifier")
	}
	for _, span := range endedSpans {
		for _, item := range span.Attributes() {
			if strings.Contains(item.Value.Emit(), "person@example.test") {
				t.Fatal("trace contained an unsafe high-cardinality value")
			}
		}
	}

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	observedNames := make(map[string]bool)
	for _, scope := range metrics.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name == "http.server.request.count" || metric.Name == "http.server.request.duration" {
				observedNames[metric.Name] = true
				assertBoundedMetricAttributes(t, metric.Data)
			}
		}
	}
	if !observedNames["http.server.request.count"] || !observedNames["http.server.request.duration"] {
		t.Fatalf("RED metrics missing: %v", observedNames)
	}
}

func assertBoundedMetricAttributes(t *testing.T, data any) {
	t.Helper()
	sets := make([]attribute.Set, 0, 1)
	switch typed := data.(type) {
	case metricdata.Sum[int64]:
		for _, point := range typed.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Histogram[float64]:
		for _, point := range typed.DataPoints {
			sets = append(sets, point.Attributes)
		}
	default:
		t.Fatalf("unexpected RED aggregation %T", data)
	}
	if len(sets) != 1 {
		t.Fatalf("metric points = %d, want 1", len(sets))
	}
	want := map[string]string{
		"http.request.method": "GET", "http.route": "/health/ready", "atlas.outcome": "ok", "http.response.status_code": "200",
	}
	for _, item := range sets[0].ToSlice() {
		value, allowed := want[string(item.Key)]
		if !allowed || item.Value.Emit() != value {
			t.Fatalf("unbounded or unexpected metric attribute: %s=%s", item.Key, item.Value.Emit())
		}
		delete(want, string(item.Key))
	}
	if len(want) != 0 {
		t.Fatalf("metric attributes missing: %v", want)
	}
}

func TestInvalidRequestMetadataIsReplacedNotReflected(t *testing.T) {
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, nil)
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	request.Header["X-Request-Id"] = []string{"person@example.test", validRequestID}
	request.Header.Set("X-Correlation-Id", "person@example.test")
	request.Header.Set("traceparent", "person@example.test")
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	for _, name := range []string{"X-Request-Id", "X-Correlation-Id", "traceparent"} {
		if strings.Contains(response.Header().Get(name), "person@example") {
			t.Fatalf("%s reflected unsafe input", name)
		}
	}
	if response.Header().Get("X-Request-Id") != "req_00000000000000000001" {
		t.Fatal("duplicate request IDs were trusted")
	}
}

func TestIdentifierAndLoggingDegradationRemainSafe(t *testing.T) {
	calls := 0
	generator := func(prefix string) (identifier.ID, error) {
		calls++
		if calls > 2 {
			return identifier.ID{}, errors.New("person@example.test")
		}
		return staticID(prefix)
	}
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.NewID = generator
		options.Logs = panickingLogger{}
	})
	response := perform(app.Handler(), http.MethodGet, "/health/live", nil)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("identifier degradation status = %d", response.Code)
	}
	assertProblem(t, response, "DEPENDENCY_DEGRADED")
	if strings.Contains(response.Body.String(), "person@example") {
		t.Fatal("identifier failure leaked unsafe detail")
	}

	healthy := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Logs = panickingLogger{}
	})
	healthyResponse := perform(healthy.Handler(), http.MethodGet, "/health/ready", nil)
	if healthyResponse.Code != http.StatusOK {
		t.Fatalf("telemetry failure changed readiness response: %d", healthyResponse.Code)
	}
}

func TestReadinessCheckerReceivesBoundedDeadline(t *testing.T) {
	deadlineObserved := false
	app := newTestApp(t, ReadinessState{}, func(options *Options) {
		options.ReadinessTimeout = 250 * time.Millisecond
		options.Readiness = ReadinessFunc(func(ctx context.Context) ReadinessState {
			deadline, found := ctx.Deadline()
			deadlineObserved = found && time.Until(deadline) <= 250*time.Millisecond && time.Until(deadline) > 0
			return ReadinessState{DependenciesReady: true, MigrationsCurrent: true}
		})
	})
	response := perform(app.Handler(), http.MethodGet, "/health/ready", nil)
	if response.Code != http.StatusOK || !deadlineObserved {
		t.Fatalf("readiness deadline not enforced: status=%d observed=%v", response.Code, deadlineObserved)
	}
}

func TestSecureCORSMatrix(t *testing.T) {
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.CORS = CORSConfig{AllowedOrigins: []string{"https://wallet.example.test"}, AllowCredentials: true}
	})

	allowed := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	allowed.Header.Set("Origin", "https://wallet.example.test")
	allowedResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(allowedResponse, allowed)
	if allowedResponse.Header().Get("Access-Control-Allow-Origin") != "https://wallet.example.test" ||
		allowedResponse.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("exact allowed origin was not returned")
	}

	denied := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	denied.Header.Set("Origin", "https://evil.example.test")
	deniedResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusOK || deniedResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("denied actual origin received CORS permission")
	}

	preflight := httptest.NewRequest(http.MethodOptions, "/health/live", nil)
	preflight.Header.Set("Origin", "https://wallet.example.test")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight.Header.Set("Access-Control-Request-Headers", "traceparent, X-Request-Id")
	preflightResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent || preflightResponse.Header().Get("Access-Control-Allow-Origin") == "*" {
		t.Fatalf("preflight status/origin = %d/%q", preflightResponse.Code, preflightResponse.Header().Get("Access-Control-Allow-Origin"))
	}

	deniedPreflight := httptest.NewRequest(http.MethodOptions, "/health/live", nil)
	deniedPreflight.Header.Set("Origin", "https://evil.example.test")
	deniedPreflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	deniedPreflightResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(deniedPreflightResponse, deniedPreflight)
	assertProblem(t, deniedPreflightResponse, "CORS_ORIGIN_DENIED")

	_, err := New(Options{
		Build:     BuildInfo{SourceRevision: "development", ContractVersion: ContractVersion, BuildTime: testBuildTime},
		Readiness: ReadinessFunc(func(context.Context) ReadinessState { return ReadinessState{} }),
		CORS:      CORSConfig{AllowedOrigins: []string{"*"}, AllowCredentials: true},
	})
	if err == nil {
		t.Fatal("wildcard credentialed CORS was accepted")
	}
}

func TestResourceLimitsRouteInventoryAndSafeProblems(t *testing.T) {
	secret := "person@example.test postgres://internal"
	app := newTestApp(t, ReadinessState{}, func(options *Options) {
		options.MaxBodyBytes = 32
		options.Readiness = ReadinessFunc(func(context.Context) ReadinessState { panic(secret) })
	})

	tests := []struct {
		name       string
		request    *http.Request
		wantStatus int
		wantCode   string
	}{
		{name: "declared large body", request: httptest.NewRequest(http.MethodGet, "/health/live", strings.NewReader(strings.Repeat("x", 33))), wantStatus: 413, wantCode: "REQUEST_TOO_LARGE"},
		{name: "chunked large body", request: func() *http.Request {
			r := httptest.NewRequest(http.MethodGet, "/health/live", strings.NewReader(strings.Repeat("x", 33)))
			r.ContentLength = -1
			return r
		}(), wantStatus: 413, wantCode: "REQUEST_TOO_LARGE"},
		{name: "small unexpected body", request: httptest.NewRequest(http.MethodGet, "/health/live", strings.NewReader("{}")), wantStatus: 400, wantCode: "REQUEST_MALFORMED"},
		{name: "compressed body", request: func() *http.Request {
			r := httptest.NewRequest(http.MethodGet, "/health/live", nil)
			r.Header.Set("Content-Encoding", "gzip")
			return r
		}(), wantStatus: 415, wantCode: "UNSUPPORTED_MEDIA_TYPE"},
		{name: "query", request: httptest.NewRequest(http.MethodGet, "/health/live?debug=true", nil), wantStatus: 400, wantCode: "REQUEST_MALFORMED"},
		{name: "unknown route", request: httptest.NewRequest(http.MethodGet, "/debug/vars", nil), wantStatus: 404, wantCode: "ROUTE_NOT_FOUND"},
		{name: "method", request: httptest.NewRequest(http.MethodPost, "/version", nil), wantStatus: 405, wantCode: "METHOD_NOT_ALLOWED"},
		{name: "panic", request: httptest.NewRequest(http.MethodGet, "/health/ready", nil), wantStatus: 500, wantCode: "INTERNAL_ERROR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			app.Handler().ServeHTTP(response, test.request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body)
			}
			assertProblem(t, response, test.wantCode)
			if strings.Contains(response.Body.String(), secret) || strings.Contains(response.Body.String(), "postgres") {
				t.Fatal("problem response leaked internal detail")
			}
		})
	}

	wantRoutes := []string{"/health/live", "/health/ready", "/version"}
	if !reflect.DeepEqual(foundationRoutes, wantRoutes) {
		t.Fatalf("route inventory = %v, want %v", foundationRoutes, wantRoutes)
	}
}

func assertProblem(t *testing.T, response *httptest.ResponseRecorder, wantCode string) problemResponse {
	t.Helper()
	if response.Header().Get("Content-Type") != "application/problem+json" {
		t.Errorf("content type = %q", response.Header().Get("Content-Type"))
	}
	var problem problemResponse
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Code != wantCode || problem.Status != response.Code || problem.RequestID == "" || problem.CorrelationID == "" {
		t.Fatalf("problem = %+v, want code=%s/status=%d", problem, wantCode, response.Code)
	}
	return problem
}

func TestBuildAndHTTPConfigurationFailClosed(t *testing.T) {
	base := Options{
		Build:     BuildInfo{SourceRevision: "development", ContractVersion: ContractVersion, BuildTime: testBuildTime},
		Readiness: ReadinessFunc(func(context.Context) ReadinessState { return ReadinessState{} }),
	}
	for _, mutate := range []func(*Options){
		func(options *Options) { options.Build.SourceRevision = "main\nperson@example.test" },
		func(options *Options) { options.Build.ContractVersion = "latest" },
		func(options *Options) { options.Build.BuildTime = time.Time{} },
		func(options *Options) { options.MaxBodyBytes = 9 << 20 },
	} {
		options := base
		mutate(&options)
		if _, err := New(options); err == nil {
			t.Fatal("unsafe application configuration was accepted")
		}
	}

	config := DefaultHTTPConfig("127.0.0.1:0")
	server, err := NewHTTPServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), config)
	if err != nil {
		t.Fatal(err)
	}
	if server.ReadHeaderTimeout != 5*time.Second || server.ReadTimeout != 10*time.Second || server.WriteTimeout != 15*time.Second ||
		server.IdleTimeout != 60*time.Second || server.MaxHeaderBytes != 16<<10 {
		t.Fatalf("HTTP safety defaults changed: %+v", server)
	}
	config.ReadTimeout = 0
	if _, err := NewHTTPServer(http.NotFoundHandler(), config); err == nil {
		t.Fatal("zero HTTP deadline was accepted")
	}
}

func TestSlowHeaderIsBoundedByServerDeadline(t *testing.T) {
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, nil)
	config := DefaultHTTPConfig("127.0.0.1:0")
	config.ReadHeaderTimeout = 75 * time.Millisecond
	config.ReadTimeout = 150 * time.Millisecond
	server, err := NewHTTPServer(app.Handler(), config)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()

	connection, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := io.WriteString(connection, "GET /health/live HTTP/1.1\r\nHost:"); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	buffer := make([]byte, 256)
	_, readErr := connection.Read(buffer)
	if readErr != nil {
		var netError net.Error
		if errors.As(readErr, &netError) && netError.Timeout() {
			t.Fatalf("server did not enforce header deadline: %v", readErr)
		}
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("slow header held connection for %s", elapsed)
	}

	shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	if err := <-serveDone; !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve() = %v", err)
	}
}

func FuzzUntrustedRequestMetadata(f *testing.F) {
	for _, seed := range [][3]string{
		{validRequestID, validCorrelationID, validTraceparent},
		{"person@example.test", "cor_01JATLAS00000000000001", "invalid"},
		{"", "", ""},
	} {
		f.Add(seed[0], seed[1], seed[2])
	}
	f.Fuzz(func(t *testing.T, requestID, correlationID, traceparent string) {
		app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, nil)
		request := httptest.NewRequest(http.MethodGet, "/health/live", bytes.NewReader(nil))
		request.Header.Set("X-Request-Id", requestID)
		request.Header.Set("X-Correlation-Id", correlationID)
		request.Header.Set("traceparent", traceparent)
		response := httptest.NewRecorder()
		app.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d", response.Code)
		}
		for name, prefix := range map[string]string{"X-Request-Id": "req", "X-Correlation-Id": "cor"} {
			parsed, err := identifier.Parse(response.Header().Get(name))
			if err != nil || parsed.Prefix() != prefix {
				t.Fatalf("unsafe %s response: %q", name, response.Header().Get(name))
			}
		}
		if _, _, _, ok := parseTraceparent(response.Header().Get("traceparent")); !ok {
			t.Fatalf("unsafe trace response: %q", response.Header().Get("traceparent"))
		}
	})
}
