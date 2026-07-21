// Command simulator owns the feature-free provider-simulator process lifecycle.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/environment"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "atlas simulator stopped")
		os.Exit(1)
	}
}

func run() error {
	path := os.Getenv("ATLAS_ENV_CONFIG")
	if _, err := environment.Load(path, time.Now().UTC()); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	return nil
}
