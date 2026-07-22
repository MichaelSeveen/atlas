package environment

import (
	"context"
	"net"
	"sync"
	"time"
)

type ProbeSet struct {
	services []Service
	dialer   net.Dialer
}

func NewProbeSet(config Config) ProbeSet {
	critical := make([]Service, 0, len(config.Services))
	for _, service := range config.Services {
		// Telemetry is intentionally non-authoritative. Export failure is an
		// operational degradation, never an API-readiness dependency.
		if service.Name == "telemetry" {
			continue
		}
		critical = append(critical, service)
	}
	return ProbeSet{
		services: critical,
		dialer:   net.Dialer{Timeout: 500 * time.Millisecond},
	}
}

func (p ProbeSet) Ready(ctx context.Context) bool {
	if len(p.services) == 0 {
		return false
	}
	results := make(chan bool, len(p.services))
	var wait sync.WaitGroup
	for _, service := range p.services {
		wait.Add(1)
		go func(address string) {
			defer wait.Done()
			connection, err := p.dialer.DialContext(ctx, "tcp", address)
			if err != nil {
				results <- false
				return
			}
			_ = connection.Close()
			results <- true
		}(service.Address)
	}
	wait.Wait()
	close(results)
	for ready := range results {
		if !ready {
			return false
		}
	}
	return true
}

func (c Config) MigrationsCurrent() bool {
	return c.MigrationState == RequiredMigrationState
}
