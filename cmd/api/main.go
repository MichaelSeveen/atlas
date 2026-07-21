// Command api runs the Atlas synchronous API/BFF process foundation.
// It exposes operational health/version endpoints only; it has no product API.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MichaelSeveen/atlas/cmd/api/internal/server"
	"github.com/MichaelSeveen/atlas/internal/platform/database"
	"github.com/MichaelSeveen/atlas/internal/platform/environment"
	"github.com/MichaelSeveen/atlas/internal/platform/logging"
	"github.com/MichaelSeveen/atlas/internal/platform/telemetry"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	traceapi "go.opentelemetry.io/otel/trace"
)

var (
	sourceRevision  = "development"
	contractVersion = server.ContractVersion
	buildTime       = "1970-01-01T00:00:00Z"
)

func main() {
	if err := run(); err != nil {
		_ = logging.RecordProcessFailure(os.Stderr, "api", sourceRevision, time.Now().UTC())
		os.Exit(1)
	}
}

func run() error {
	builtAt, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		return errors.New("invalid build metadata")
	}

	baseContext := context.Background()
	config, err := loadEnvironment(os.Getenv("ATLAS_ENV_CONFIG"))
	if err != nil {
		return err
	}
	var telemetryRuntime *telemetry.Runtime
	var tracer traceapi.Tracer
	var meter metricapi.Meter
	var propagator propagation.TextMapPropagator
	if config != nil {
		telemetryRuntime, err = telemetry.NewForEnvironment(baseContext, "atlas-api", sourceRevision, builtAt, *config)
		if err != nil {
			return err
		}
		tracer = telemetryRuntime.Tracer()
		meter = telemetryRuntime.Meter()
		propagator = telemetryRuntime.Propagator()
		defer func() {
			shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = telemetryRuntime.Shutdown(shutdown)
		}()
	}
	readiness, cors, closeReadiness, err := environmentOptions(baseContext, config, database.ProbeOptions{Tracer: tracer, Meter: meter})
	if err != nil {
		return err
	}
	defer closeReadiness()
	logger, err := logging.NewJSONRecorder(os.Stdout)
	if err != nil {
		return err
	}
	app, err := server.New(server.Options{
		Build: server.BuildInfo{
			SourceRevision:  sourceRevision,
			ContractVersion: contractVersion,
			BuildTime:       builtAt,
		},
		Readiness:  readiness,
		CORS:       cors,
		Tracer:     tracer,
		Meter:      meter,
		Propagator: propagator,
		Logs:       logger,
	})
	if err != nil {
		return err
	}
	_ = logger.Record(logging.Record{
		Timestamp: time.Now().UTC(), Event: logging.EventProcessStarted, Severity: logging.SeverityInfo,
		Module: "api", Outcome: "started", SourceRevision: sourceRevision,
	})
	defer func() {
		_ = logger.Record(logging.Record{
			Timestamp: time.Now().UTC(), Event: logging.EventProcessStopped, Severity: logging.SeverityInfo,
			Module: "api", Outcome: "stopped", SourceRevision: sourceRevision,
		})
	}()

	address := os.Getenv("ATLAS_HTTP_ADDR")
	if address == "" {
		address = "127.0.0.1:8080"
	}
	httpConfig := server.DefaultHTTPConfig(address)
	httpServer, err := server.NewHTTPServer(app.Handler(), httpConfig)
	if err != nil {
		return err
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownContext.Done()
		deadline, cancel := context.WithTimeout(context.Background(), httpConfig.ShutdownTimeout)
		defer cancel()
		_ = httpServer.Shutdown(deadline)
	}()

	err = httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func loadEnvironment(path string) (*environment.Config, error) {
	if path == "" {
		return nil, nil
	}
	config, err := environment.Load(path, time.Now().UTC())
	if err != nil {
		return nil, errors.New("invalid environment configuration")
	}
	return &config, nil
}

func environmentOptions(ctx context.Context, config *environment.Config, probeOptions database.ProbeOptions) (server.ReadinessChecker, server.CORSConfig, func(), error) {
	if config == nil {
		return server.ReadinessFunc(func(context.Context) server.ReadinessState {
			// A standalone process remains deliberately not ready. The S04 local
			// environment supplies a validated configuration and dependency probes.
			return server.ReadinessState{}
		}), server.CORSConfig{}, func() {}, nil
	}
	probes := environment.NewProbeSet(*config)
	databaseConfig, err := database.ConfigFromEnvironment()
	if err != nil {
		return nil, server.CORSConfig{}, nil, errors.New("invalid database readiness configuration")
	}
	schemaProbe, err := database.NewSchemaProbe(ctx, databaseConfig, probeOptions)
	if err != nil {
		return nil, server.CORSConfig{}, nil, errors.New("invalid database readiness configuration")
	}
	return server.ReadinessFunc(func(ctx context.Context) server.ReadinessState {
		return server.ReadinessState{
			DependenciesReady: probes.Ready(ctx),
			MigrationsCurrent: schemaProbe.Ready(ctx),
		}
	}), server.CORSConfig{AllowedOrigins: config.AllowedOrigins}, schemaProbe.Close, nil
}
