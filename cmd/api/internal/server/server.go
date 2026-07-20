// Package server implements the feature-free Atlas HTTP foundation.
package server

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"regexp"
	"sync"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/clock"
	"github.com/MichaelSeveen/atlas/internal/platform/correlation"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

const ContractVersion = "2026-07-20"

var sourceRevisionPattern = regexp.MustCompile(`^(development|[0-9a-f]{7,64})$`)

type BuildInfo struct {
	SourceRevision  string
	ContractVersion string
	BuildTime       time.Time
}

type ReadinessState struct {
	DependenciesReady bool
	MigrationsCurrent bool
}

func (s ReadinessState) Ready() bool {
	return s.DependenciesReady && s.MigrationsCurrent
}

type ReadinessChecker interface {
	Check(context.Context) ReadinessState
}

type ReadinessFunc func(context.Context) ReadinessState

func (f ReadinessFunc) Check(ctx context.Context) ReadinessState {
	return f(ctx)
}

type IDGenerator func(prefix string) (identifier.ID, error)

type Options struct {
	Build            BuildInfo
	Readiness        ReadinessChecker
	Clock            clock.Clock
	NewID            IDGenerator
	Entropy          io.Reader
	Traces           TraceRecorder
	Metrics          MetricsRecorder
	CORS             CORSConfig
	MaxBodyBytes     int64
	ReadinessTimeout time.Duration
}

type App struct {
	build            BuildInfo
	readiness        ReadinessChecker
	clock            clock.Clock
	newID            IDGenerator
	entropy          io.Reader
	entropyMu        sync.Mutex
	traces           TraceRecorder
	metrics          MetricsRecorder
	cors             corsPolicy
	maxBodyBytes     int64
	readinessTimeout time.Duration
	emergencyContext correlation.Context
}

func New(options Options) (*App, error) {
	if !sourceRevisionPattern.MatchString(options.Build.SourceRevision) {
		return nil, errors.New("invalid source revision metadata")
	}
	if options.Build.ContractVersion != ContractVersion {
		return nil, errors.New("contract version does not match the canonical API")
	}
	if options.Build.BuildTime.IsZero() {
		return nil, errors.New("build time is required")
	}
	options.Build.BuildTime = options.Build.BuildTime.UTC()

	if options.Readiness == nil {
		return nil, errors.New("readiness checker is required")
	}
	if options.Clock == nil {
		options.Clock = clock.System{}
	}
	if options.NewID == nil {
		options.NewID = identifier.New
	}
	if options.Entropy == nil {
		options.Entropy = rand.Reader
	}
	if options.Traces == nil {
		options.Traces = discardTraceRecorder{}
	}
	if options.Metrics == nil {
		options.Metrics = discardMetricsRecorder{}
	}
	if options.MaxBodyBytes == 0 {
		options.MaxBodyBytes = 1 << 20
	}
	if options.MaxBodyBytes < 1 || options.MaxBodyBytes > 8<<20 {
		return nil, errors.New("request body limit is outside the foundation policy")
	}
	if options.ReadinessTimeout == 0 {
		options.ReadinessTimeout = 2 * time.Second
	}
	if options.ReadinessTimeout < 10*time.Millisecond || options.ReadinessTimeout > 5*time.Second {
		return nil, errors.New("readiness timeout is outside the foundation policy")
	}
	cors, err := newCORSPolicy(options.CORS)
	if err != nil {
		return nil, err
	}

	emergencyRequestID, err := generatedID(options.NewID, "req")
	if err != nil {
		return nil, errors.New("request identifier generator is unavailable")
	}
	emergencyCorrelationID, err := generatedID(options.NewID, "cor")
	if err != nil {
		return nil, errors.New("correlation identifier generator is unavailable")
	}
	emergencyContext, err := correlation.New(emergencyRequestID, emergencyCorrelationID)
	if err != nil {
		return nil, errors.New("emergency request context is invalid")
	}

	return &App{
		build:            options.Build,
		readiness:        options.Readiness,
		clock:            options.Clock,
		newID:            options.NewID,
		entropy:          options.Entropy,
		traces:           options.Traces,
		metrics:          options.Metrics,
		cors:             cors,
		maxBodyBytes:     options.MaxBodyBytes,
		readinessTimeout: options.ReadinessTimeout,
		emergencyContext: emergencyContext,
	}, nil
}
