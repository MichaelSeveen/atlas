package environment

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTelemetryOutageDoesNotDetermineReadiness(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	config := Config{Services: []Service{
		{Name: "postgres", Address: listener.Addr().String(), Synthetic: true},
		{Name: "telemetry", Address: "127.0.0.1:1", Synthetic: true},
	}}
	probe := NewProbeSet(config)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !probe.Ready(ctx) {
		t.Fatal("unavailable telemetry incorrectly failed authoritative dependency readiness")
	}
}

func TestProbeSetStillFailsClosedWithoutAuthoritativeDependencies(t *testing.T) {
	probe := NewProbeSet(Config{Services: []Service{{Name: "telemetry", Address: "127.0.0.1:1", Synthetic: true}}})
	if probe.Ready(context.Background()) {
		t.Fatal("telemetry-only topology was reported ready")
	}
}
