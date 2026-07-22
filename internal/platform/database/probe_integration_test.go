package database

import (
	"context"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestMostAgentsSkip10ConstrainedDatabasePool(t *testing.T) {
	if os.Getenv("ATLAS_S08_DATABASE_INTEGRATION") != "1" {
		t.Skip("set ATLAS_S08_DATABASE_INTEGRATION=1 with the migrated local PostgreSQL foundation")
	}
	config, err := ConfigFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	probe, err := NewSchemaProbe(context.Background(), config, ProbeOptions{MaxConnections: 1})
	if err != nil {
		t.Fatal(err)
	}

	const goroutines = 24
	start := make(chan struct{})
	results := make(chan bool, goroutines)
	var group sync.WaitGroup
	group.Add(goroutines)
	for range goroutines {
		go func() {
			defer group.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			results <- probe.Ready(ctx)
		}()
	}

	close(start)
	completed := make(chan struct{})
	go func() {
		group.Wait()
		close(results)
		close(completed)
	}()
	select {
	case <-completed:
	case <-time.After(15 * time.Second):
		t.Fatal("concurrent readiness probes did not complete within the bounded deadline")
	}
	for ready := range results {
		if !ready {
			t.Fatal("migrated database became unavailable under the constrained pool")
		}
	}
	if got := probe.pool.Stat().MaxConns(); got != 1 {
		t.Fatalf("constrained test used %d connections, want 1", got)
	}

	probe.Close()
	deadline := time.Now().Add(2 * time.Second)
	for probe.pool.Stat().TotalConns() != 0 && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	if got := probe.pool.Stat().TotalConns(); got != 0 {
		t.Fatalf("readiness pool retained %d connections after close", got)
	}
}
