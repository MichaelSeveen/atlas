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
	return ProbeSet{
		services: append([]Service(nil), config.Services...),
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
