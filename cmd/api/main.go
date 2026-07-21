// Command api runs the Atlas synchronous API/BFF process foundation.
// S03 exposes operational health/version endpoints only; it has no product API.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MichaelSeveen/atlas/cmd/api/internal/server"
	"github.com/MichaelSeveen/atlas/internal/platform/environment"
)

var (
	sourceRevision  = "development"
	contractVersion = server.ContractVersion
	buildTime       = "1970-01-01T00:00:00Z"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "atlas api stopped")
		os.Exit(1)
	}
}

func run() error {
	builtAt, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		return errors.New("invalid build metadata")
	}

	readiness, cors, err := environmentOptions(os.Getenv("ATLAS_ENV_CONFIG"))
	if err != nil {
		return err
	}
	app, err := server.New(server.Options{
		Build: server.BuildInfo{
			SourceRevision:  sourceRevision,
			ContractVersion: contractVersion,
			BuildTime:       builtAt,
		},
		Readiness: readiness,
		CORS:      cors,
	})
	if err != nil {
		return err
	}

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

func environmentOptions(path string) (server.ReadinessChecker, server.CORSConfig, error) {
	if path == "" {
		return server.ReadinessFunc(func(context.Context) server.ReadinessState {
			// A standalone process remains deliberately not ready. The S04 local
			// environment supplies a validated configuration and dependency probes.
			return server.ReadinessState{}
		}), server.CORSConfig{}, nil
	}
	config, err := environment.Load(path, time.Now().UTC())
	if err != nil {
		return nil, server.CORSConfig{}, errors.New("invalid environment configuration")
	}
	probes := environment.NewProbeSet(config)
	return server.ReadinessFunc(func(ctx context.Context) server.ReadinessState {
		return server.ReadinessState{
			DependenciesReady: probes.Ready(ctx),
			MigrationsCurrent: config.MigrationsCurrent(),
		}
	}), server.CORSConfig{AllowedOrigins: config.AllowedOrigins}, nil
}
