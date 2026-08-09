// Package ratelimit implements reconstructible credential abuse counters.
package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/MichaelSeveen/atlas/internal/identity"
)

const localCounterCapacity = identity.CredentialProcessLocalCounterCapacity

var incrementWindows = redis.NewScript(`
local one = redis.call('INCR', KEYS[1])
if one == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
local two = redis.call('INCR', KEYS[2])
if two == 1 then redis.call('EXPIRE', KEYS[2], ARGV[1]) end
local three = redis.call('INCR', KEYS[3])
if three == 1 then redis.call('EXPIRE', KEYS[3], ARGV[1]) end
return {one, two, three}
`)

type localCounter struct {
	count     int
	expiresAt time.Time
}

// Limiter prefers Redis and falls back to a stricter bounded in-process
// window. Neither path carries authorization truth.
type Limiter struct {
	client *redis.Client

	mu       sync.Mutex
	counters map[string]localCounter
}

func New(address string, password string) (*Limiter, error) {
	if address == "" || password == "" {
		return nil, errors.New("credential rate limiter configuration is incomplete")
	}
	return &Limiter{
		client: redis.NewClient(&redis.Options{
			Addr: address, Password: password, DB: 0,
			DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second,
			MaxRetries: 1, PoolSize: 4,
		}),
		counters: make(map[string]localCounter),
	}, nil
}

func (limiter *Limiter) Close() error {
	if limiter == nil || limiter.client == nil {
		return nil
	}
	return limiter.client.Close()
}

func (limiter *Limiter) Allow(
	ctx context.Context,
	subject identity.CredentialRateSubject,
	now time.Time,
) (identity.CredentialRateDecision, error) {
	if limiter == nil || limiter.client == nil || subject.CredentialID.IsZero() ||
		subject.TenantID.IsZero() || now.IsZero() {
		return identity.CredentialRateDecision{}, errors.New("credential rate subject is invalid")
	}
	keys := []string{
		"atlas:credential-rate:v1:key:" + digestString(subject.CredentialID.String()),
		"atlas:credential-rate:v1:tenant:" + digestString(subject.TenantID.String()),
		"atlas:credential-rate:v1:network:" + hex.EncodeToString(subject.NetworkSignal[:]),
	}
	result, err := incrementWindows.Run(
		ctx, limiter.client, keys, strconv.Itoa(int(identity.CredentialRateWindow/time.Second)),
	).Int64Slice()
	if err == nil && len(result) == 3 {
		return identity.CredentialRateDecision{
			Allowed: result[0] <= identity.CredentialRatePerKey &&
				result[1] <= identity.CredentialRatePerTenant &&
				result[2] <= identity.CredentialRatePerNetwork,
		}, nil
	}
	return limiter.allowLocal(keys, now), nil
}

func (limiter *Limiter) allowLocal(keys []string, now time.Time) identity.CredentialRateDecision {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	for key, counter := range limiter.counters {
		if !now.Before(counter.expiresAt) {
			delete(limiter.counters, key)
		}
	}
	missing := 0
	for _, key := range keys {
		if _, exists := limiter.counters[key]; !exists {
			missing++
		}
	}
	if len(limiter.counters)+missing > localCounterCapacity {
		return identity.CredentialRateDecision{Allowed: false, Fallback: true}
	}
	counts := make([]int, len(keys))
	for index, key := range keys {
		counter := limiter.counters[key]
		if counter.expiresAt.IsZero() || !now.Before(counter.expiresAt) {
			counter = localCounter{expiresAt: now.Add(identity.CredentialRateWindow)}
		}
		counter.count++
		limiter.counters[key] = counter
		counts[index] = counter.count
	}
	return identity.CredentialRateDecision{
		Allowed: counts[0] <= identity.CredentialFallbackRatePerKey &&
			counts[1] <= identity.CredentialFallbackRatePerTenant &&
			counts[2] <= identity.CredentialFallbackRatePerNetwork,
		Fallback: true,
	}
}

func digestString(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

var _ identity.CredentialRateLimiter = (*Limiter)(nil)
