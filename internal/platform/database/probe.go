// Package database owns feature-free PostgreSQL connectivity and schema readiness.
package database

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/migration"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	metricapi "go.opentelemetry.io/otel/metric"
	traceapi "go.opentelemetry.io/otel/trace"
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,62}$`)

type Config struct {
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
}

type SchemaProbe struct {
	pool              *pgxpool.Pool
	tracer            traceapi.Tracer
	readinessCounter  metricapi.Int64Counter
	readinessDuration metricapi.Float64Histogram
	poolConnections   metricapi.Int64Gauge
}

type ProbeOptions struct {
	Tracer traceapi.Tracer
	Meter  metricapi.Meter
}

func ConfigFromEnvironment() (Config, error) {
	port, err := strconv.ParseUint(os.Getenv("ATLAS_POSTGRES_PORT"), 10, 16)
	if err != nil {
		return Config{}, errors.New("database port is invalid")
	}
	config := Config{
		Host: os.Getenv("ATLAS_POSTGRES_HOST"), Port: uint16(port), Database: os.Getenv("ATLAS_POSTGRES_DB"),
		User: os.Getenv("ATLAS_POSTGRES_API_USER"), Password: os.Getenv("ATLAS_POSTGRES_API_PASSWORD"),
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" || strings.ContainsAny(c.Host, " /\\@") || c.Port == 0 {
		return errors.New("database endpoint is invalid")
	}
	if !identifierPattern.MatchString(c.Database) || !identifierPattern.MatchString(c.User) {
		return errors.New("database identity is invalid")
	}
	if len(c.Password) < 32 || len(c.Password) > 256 {
		return errors.New("database credential is invalid")
	}
	return nil
}

func NewSchemaProbe(ctx context.Context, config Config, options ProbeOptions) (*SchemaProbe, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	endpoint := &url.URL{
		Scheme: "postgres", User: url.UserPassword(config.User, config.Password),
		Host: net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))), Path: config.Database,
	}
	query := endpoint.Query()
	query.Set("sslmode", "disable")
	endpoint.RawQuery = query.Encode()
	poolConfig, err := pgxpool.ParseConfig(endpoint.String())
	if err != nil {
		return nil, errors.New("database readiness configuration is invalid")
	}
	poolConfig.ConnConfig.ConnectTimeout = 500 * time.Millisecond
	poolConfig.ConnConfig.RuntimeParams["application_name"] = "atlas-api-readiness"
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = "500"
	poolConfig.ConnConfig.RuntimeParams["lock_timeout"] = "250"
	poolConfig.MinConns = 0
	poolConfig.MaxConns = 4
	poolConfig.MaxConnIdleTime = 2 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.New("create database readiness pool")
	}
	probe := &SchemaProbe{pool: pool, tracer: options.Tracer}
	if options.Meter != nil {
		probe.readinessCounter, err = options.Meter.Int64Counter("atlas.database.readiness.count",
			metricapi.WithDescription("Completed foundation schema-readiness probes."), metricapi.WithUnit("{probe}"))
		if err == nil {
			probe.readinessDuration, err = options.Meter.Float64Histogram("atlas.database.readiness.duration",
				metricapi.WithDescription("Foundation schema-readiness probe duration."), metricapi.WithUnit("s"))
		}
		if err == nil {
			probe.poolConnections, err = options.Meter.Int64Gauge("atlas.database.pool.connections",
				metricapi.WithDescription("Foundation API readiness-pool connections by bounded state."), metricapi.WithUnit("{connection}"))
		}
		if err != nil {
			pool.Close()
			return nil, errors.New("create database readiness instruments")
		}
	}
	return probe, nil
}

func (p *SchemaProbe) Ready(ctx context.Context) bool {
	if p == nil || p.pool == nil {
		return false
	}
	probeContext, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	var span traceapi.Span
	if p.tracer != nil {
		probeContext, span = p.tracer.Start(probeContext, "database.schema_readiness",
			traceapi.WithSpanKind(traceapi.SpanKindClient),
			traceapi.WithAttributes(
				attribute.String("db.system.name", "postgresql"),
				attribute.String("db.operation.name", "SELECT"),
				attribute.String("db.namespace", "atlas_foundation"),
			),
		)
		defer span.End()
	}
	started := time.Now()
	var checksum string
	err := p.pool.QueryRow(probeContext,
		"SELECT checksum FROM atlas_foundation.schema_migrations WHERE version = $1",
		migration.CurrentVersion,
	).Scan(&checksum)
	ready := err == nil && checksum == migration.CurrentChecksum
	outcome := "not_ready"
	if ready {
		outcome = "ready"
	}
	attributes := []attribute.KeyValue{attribute.String("atlas.outcome", outcome)}
	safeCounterAdd(p.readinessCounter, probeContext, 1, metricapi.WithAttributes(attributes...))
	safeHistogramRecord(p.readinessDuration, probeContext, time.Since(started).Seconds(), metricapi.WithAttributes(attributes...))
	p.recordPool(probeContext)
	if span != nil {
		span.SetAttributes(attributes...)
		if !ready {
			span.SetStatus(codes.Error, outcome)
		}
	}
	return ready
}

func (p *SchemaProbe) recordPool(ctx context.Context) {
	if p.poolConnections == nil {
		return
	}
	stats := p.pool.Stat()
	for state, value := range map[string]int64{
		"acquired": int64(stats.AcquiredConns()),
		"idle":     int64(stats.IdleConns()),
		"total":    int64(stats.TotalConns()),
		"maximum":  int64(stats.MaxConns()),
	} {
		safeGaugeRecord(p.poolConnections, ctx, value, metricapi.WithAttributes(attribute.String("state", state)))
	}
}

func (p *SchemaProbe) Close() {
	if p != nil && p.pool != nil {
		p.pool.Close()
	}
}

func safeCounterAdd(counter metricapi.Int64Counter, ctx context.Context, value int64, options ...metricapi.AddOption) {
	if counter == nil {
		return
	}
	defer func() { _ = recover() }()
	counter.Add(ctx, value, options...)
}

func safeHistogramRecord(histogram metricapi.Float64Histogram, ctx context.Context, value float64, options ...metricapi.RecordOption) {
	if histogram == nil {
		return
	}
	defer func() { _ = recover() }()
	histogram.Record(ctx, value, options...)
}

func safeGaugeRecord(gauge metricapi.Int64Gauge, ctx context.Context, value int64, options ...metricapi.RecordOption) {
	if gauge == nil {
		return
	}
	defer func() { _ = recover() }()
	gauge.Record(ctx, value, options...)
}
