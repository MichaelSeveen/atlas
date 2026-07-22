package telemetry

import (
	"context"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		ServiceName: "atlas-api", SourceRevision: "0123456", DeploymentEnvironment: "local",
		Endpoint: "127.0.0.1:1", Insecure: true,
		BuildTime: time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC), ExportInterval: 100 * time.Millisecond,
	}
}

func TestRuntimeConfigurationFailsClosed(t *testing.T) {
	tests := []func(*Config){
		func(config *Config) { config.ServiceName = "tenant-api" },
		func(config *Config) { config.SourceRevision = "latest" },
		func(config *Config) { config.DeploymentEnvironment = "production" },
		func(config *Config) { config.Endpoint = "https://collector.example" },
		func(config *Config) { config.DeploymentEnvironment = "staging" },
	}
	for _, mutate := range tests {
		config := validConfig()
		mutate(&config)
		if err := config.validate(); err == nil {
			t.Fatal("unsafe telemetry configuration was accepted")
		}
	}
}

func TestUnavailableCollectorDoesNotBlockRuntimeCreation(t *testing.T) {
	started := time.Now()
	runtime, err := New(context.Background(), validConfig())
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("collector connection blocked process startup")
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	_ = runtime.Shutdown(shutdownContext)
}
