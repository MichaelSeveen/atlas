// Command simulator owns the feature-free provider-simulator process lifecycle.
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/environment"
	"github.com/MichaelSeveen/atlas/internal/platform/logging"
	"github.com/MichaelSeveen/atlas/internal/platform/telemetry"
)

var (
	sourceRevision = "development"
	buildTime      = "1970-01-01T00:00:00Z"
)

func main() {
	if err := run(); err != nil {
		_ = logging.RecordProcessFailure(os.Stderr, "simulator", sourceRevision, time.Now().UTC())
		os.Exit(1)
	}
}

func run() error {
	builtAt, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		return errors.New("invalid build metadata")
	}
	path := os.Getenv("ATLAS_ENV_CONFIG")
	config, err := environment.Load(path, time.Now().UTC())
	if err != nil {
		return err
	}
	telemetryRuntime, err := telemetry.NewForEnvironment(context.Background(), "atlas-simulator", sourceRevision, builtAt, config)
	if err != nil {
		return err
	}
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = telemetryRuntime.Shutdown(shutdown)
	}()
	logger, err := logging.NewJSONRecorder(os.Stdout)
	if err != nil {
		return err
	}
	_ = logger.Record(logging.Record{
		Timestamp: time.Now().UTC(), Event: logging.EventProcessStarted, Severity: logging.SeverityInfo,
		Module: "simulator", Outcome: "started", SourceRevision: sourceRevision,
	})
	defer func() {
		_ = logger.Record(logging.Record{
			Timestamp: time.Now().UTC(), Event: logging.EventProcessStopped, Severity: logging.SeverityInfo,
			Module: "simulator", Outcome: "stopped", SourceRevision: sourceRevision,
		})
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	return nil
}
