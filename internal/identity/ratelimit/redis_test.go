package ratelimit

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
	"github.com/redis/go-redis/v9"
)

func TestRealRedisEnforcesCredentialTenantAndNetworkDimensions(t *testing.T) {
	address := os.Getenv("ATLAS_P01_REDIS_ADDR")
	password := os.Getenv("ATLAS_P01_REDIS_PASSWORD")
	if address == "" || password == "" {
		t.Skip("real Phase 01 Redis configuration is not present")
	}
	limiter, err := New(address, password)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = limiter.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	keys := make(map[string]struct{})
	recordKeys := func(subject identity.CredentialRateSubject) {
		keys["atlas:credential-rate:v1:key:"+digestString(subject.CredentialID.String())] = struct{}{}
		keys["atlas:credential-rate:v1:tenant:"+digestString(subject.TenantID.String())] = struct{}{}
		keys["atlas:credential-rate:v1:network:"+fmt.Sprintf("%x", subject.NetworkSignal)] = struct{}{}
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		batch := make([]string, 0, 500)
		for key := range keys {
			batch = append(batch, key)
			if len(batch) == cap(batch) {
				_ = limiter.client.Del(cleanupCtx, batch...).Err()
				batch = batch[:0]
			}
		}
		if len(batch) > 0 {
			_ = limiter.client.Del(cleanupCtx, batch...).Err()
		}
	})

	assertBoundary := func(name string, limit int, subjectFor func(int) identity.CredentialRateSubject) {
		t.Helper()
		for attempt := 1; attempt <= limit+1; attempt++ {
			subject := subjectFor(attempt)
			recordKeys(subject)
			decision, err := limiter.Allow(ctx, subject, now)
			if err != nil || decision.Fallback || decision.Allowed != (attempt <= limit) {
				t.Fatalf("%s attempt=%d decision=%+v err=%v", name, attempt, decision, err)
			}
		}
	}
	assertBoundary("credential", identity.CredentialRatePerKey, func(attempt int) identity.CredentialRateSubject {
		return identity.CredentialRateSubject{
			CredentialID:  mustRateID(t, "key", 810000),
			TenantID:      mustRateID(t, "ten", 811000+attempt),
			NetworkSignal: sha256.Sum256([]byte(fmt.Sprintf("credential-network-%d", attempt))),
		}
	})
	assertBoundary("tenant", identity.CredentialRatePerTenant, func(attempt int) identity.CredentialRateSubject {
		return identity.CredentialRateSubject{
			CredentialID:  mustRateID(t, "key", 820000+attempt),
			TenantID:      mustRateID(t, "ten", 821000),
			NetworkSignal: sha256.Sum256([]byte(fmt.Sprintf("tenant-network-%d", attempt))),
		}
	})
	fixedNetwork := sha256.Sum256([]byte("phase-01-s08-fixed-network"))
	assertBoundary("network", identity.CredentialRatePerNetwork, func(attempt int) identity.CredentialRateSubject {
		return identity.CredentialRateSubject{
			CredentialID:  mustRateID(t, "key", 830000+attempt),
			TenantID:      mustRateID(t, "ten", 831000+attempt),
			NetworkSignal: fixedNetwork,
		}
	})
}

func TestRedisOutageUsesStrictBoundedLocalLimits(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1", DialTimeout: time.Millisecond, ReadTimeout: time.Millisecond,
		WriteTimeout: time.Millisecond, MaxRetries: 0,
	})
	t.Cleanup(func() { _ = client.Close() })
	limiter := &Limiter{client: client, counters: make(map[string]localCounter)}
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	subject := identity.CredentialRateSubject{
		CredentialID: mustRateID(t, "key", 1), TenantID: mustRateID(t, "ten", 1),
		NetworkSignal: [32]byte{1},
	}
	decision, err := limiter.Allow(context.Background(), subject, now)
	if err != nil || !decision.Allowed || !decision.Fallback {
		t.Fatalf("Redis outage did not enter safe fallback: decision=%+v err=%v", decision, err)
	}

	keys := []string{"same-key", "same-tenant", "same-network"}
	local := &Limiter{counters: make(map[string]localCounter)}
	for attempt := 1; attempt <= identity.CredentialFallbackRatePerNetwork+1; attempt++ {
		decision = local.allowLocal(keys, now)
		if decision.Allowed != (attempt <= identity.CredentialFallbackRatePerNetwork) || !decision.Fallback {
			t.Fatalf("attempt %d decision=%+v", attempt, decision)
		}
	}
	decision = local.allowLocal(keys, now.Add(identity.CredentialRateWindow))
	if !decision.Allowed || !decision.Fallback {
		t.Fatalf("expired local window did not reset safely: decision=%+v", decision)
	}
}

func TestFallbackCountersAreIsolatedAndCapacityFailsClosed(t *testing.T) {
	limiter := &Limiter{counters: make(map[string]localCounter)}
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	for attempt := 0; attempt < identity.CredentialFallbackRatePerNetwork; attempt++ {
		decision := limiter.allowLocal([]string{
			fmt.Sprintf("key-%d", attempt), fmt.Sprintf("tenant-%d", attempt), "network-shared",
		}, now)
		if !decision.Allowed || !decision.Fallback {
			t.Fatalf("isolated subject %d rejected early: %+v", attempt, decision)
		}
	}
	if decision := limiter.allowLocal([]string{"key-new", "tenant-new", "network-shared"}, now); decision.Allowed {
		t.Fatal("shared network signal exceeded its strict fallback limit without rejection")
	}

	limiter.counters = make(map[string]localCounter, localCounterCapacity)
	for index := 0; index < localCounterCapacity-2; index++ {
		limiter.counters[fmt.Sprintf("nearly-occupied-%d", index)] = localCounter{
			count: 1, expiresAt: now.Add(identity.CredentialRateWindow),
		}
	}
	decision := limiter.allowLocal([]string{"three-new-key", "three-new-tenant", "three-new-network"}, now)
	if decision.Allowed || !decision.Fallback || len(limiter.counters) != localCounterCapacity-2 {
		t.Fatalf("multi-key capacity crossing did not fail closed: %+v size=%d", decision, len(limiter.counters))
	}

	limiter.counters = make(map[string]localCounter, localCounterCapacity)
	for index := 0; index < localCounterCapacity; index++ {
		limiter.counters[fmt.Sprintf("occupied-%d", index)] = localCounter{
			count: 1, expiresAt: now.Add(identity.CredentialRateWindow),
		}
	}
	decision = limiter.allowLocal([]string{"new-key", "new-tenant", "new-network"}, now)
	if decision.Allowed || !decision.Fallback || len(limiter.counters) != localCounterCapacity {
		t.Fatalf("bounded counter capacity did not fail closed: %+v size=%d", decision, len(limiter.counters))
	}
}

func mustRateID(t *testing.T, prefix string, value int) identifier.ID {
	t.Helper()
	id, err := identifier.Parse(fmt.Sprintf("%s_%020d", prefix, value))
	if err != nil {
		t.Fatal(err)
	}
	return id
}
