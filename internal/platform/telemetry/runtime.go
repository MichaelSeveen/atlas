// Package telemetry configures the OpenTelemetry foundation runtime. Export is
// deliberately best effort: collector failure must not determine readiness or
// block the process's authoritative work.
package telemetry

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/environment"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	traceapi "go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName           string
	SourceRevision        string
	DeploymentEnvironment string
	Endpoint              string
	Insecure              bool
	BuildTime             time.Time
	ExportInterval        time.Duration
}

type Runtime struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         traceapi.Tracer
	meter          metricapi.Meter
	propagator     propagation.TextMapPropagator
}

func NewForEnvironment(ctx context.Context, serviceName, sourceRevision string, buildTime time.Time, config environment.Config) (*Runtime, error) {
	endpoint := ""
	for _, service := range config.Services {
		if service.Name == "telemetry" {
			endpoint = service.Address
			break
		}
	}
	return New(ctx, Config{
		ServiceName: serviceName, SourceRevision: sourceRevision, BuildTime: buildTime,
		DeploymentEnvironment: string(config.Environment), Endpoint: endpoint,
		Insecure: config.Environment == environment.Local || config.Environment == environment.Test,
	})
}

var (
	servicePattern  = regexp.MustCompile(`^(atlas-api|atlas-worker|atlas-simulator)$`)
	revisionPattern = regexp.MustCompile(`^(development|[0-9a-f]{7,64})$`)
)

func New(ctx context.Context, config Config) (*Runtime, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	// The default SDK handler writes exporter errors, including transport
	// details, through an unstructured process-global sink. Atlas reports
	// collector availability through bounded metrics and alerts instead; raw
	// exporter errors are therefore discarded at their source boundary.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) {}))
	traceOptions := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(config.Endpoint)}
	metricOptions := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(config.Endpoint)}
	if config.Insecure {
		traceOptions = append(traceOptions, otlptracegrpc.WithInsecure())
		metricOptions = append(metricOptions, otlpmetricgrpc.WithInsecure())
	}
	traceExporter, err := otlptracegrpc.New(ctx, traceOptions...)
	if err != nil {
		return nil, errors.New("configure trace exporter")
	}
	metricExporter, err := otlpmetricgrpc.New(ctx, metricOptions...)
	if err != nil {
		return nil, errors.New("configure metric exporter")
	}
	resources, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", config.ServiceName),
		attribute.String("service.version", config.SourceRevision),
		attribute.String("deployment.environment.name", config.DeploymentEnvironment),
	))
	if err != nil {
		return nil, errors.New("configure telemetry resource")
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resources),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(500*time.Millisecond),
			sdktrace.WithExportTimeout(2*time.Second),
			sdktrace.WithMaxQueueSize(512),
		),
	)
	reader := sdkmetric.NewPeriodicReader(metricExporter,
		sdkmetric.WithInterval(config.ExportInterval),
		sdkmetric.WithTimeout(2*time.Second),
	)
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithResource(resources), sdkmetric.WithReader(reader))
	runtime := &Runtime{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		tracer:         tracerProvider.Tracer("github.com/MichaelSeveen/atlas/foundation"),
		meter:          meterProvider.Meter("github.com/MichaelSeveen/atlas/foundation"),
		propagator:     propagation.TraceContext{},
	}
	if err := runtime.recordBuild(ctx, config); err != nil {
		shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = runtime.Shutdown(shutdownContext)
		return nil, err
	}
	return runtime, nil
}

func (config *Config) validate() error {
	if !servicePattern.MatchString(config.ServiceName) || !revisionPattern.MatchString(config.SourceRevision) || config.BuildTime.IsZero() {
		return errors.New("telemetry build identity is invalid")
	}
	switch config.DeploymentEnvironment {
	case "local", "test", "staging", "production-reference":
	default:
		return errors.New("telemetry environment is invalid")
	}
	host, port, err := net.SplitHostPort(config.Endpoint)
	if err != nil || strings.TrimSpace(host) == "" {
		return errors.New("telemetry endpoint is invalid")
	}
	parsedPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil || parsedPort == 0 {
		return errors.New("telemetry endpoint is invalid")
	}
	if config.Insecure && config.DeploymentEnvironment != "local" && config.DeploymentEnvironment != "test" {
		return errors.New("insecure telemetry transport is restricted to local and test")
	}
	if config.ExportInterval == 0 {
		config.ExportInterval = 5 * time.Second
	}
	if config.ExportInterval < 100*time.Millisecond || config.ExportInterval > time.Minute {
		return errors.New("telemetry export interval is outside policy")
	}
	config.BuildTime = config.BuildTime.UTC()
	return nil
}

func (runtime *Runtime) recordBuild(ctx context.Context, config Config) error {
	gauge, err := runtime.meter.Int64Gauge("atlas.build.info",
		metricapi.WithDescription("Revision-bound process build marker."),
		metricapi.WithUnit("{build}"),
	)
	if err != nil {
		return errors.New("create build metric")
	}
	gauge.Record(ctx, 1, metricapi.WithAttributes(
		attribute.String("source.revision", config.SourceRevision),
		attribute.String("build.time", config.BuildTime.Format(time.RFC3339)),
	))
	return nil
}

func (runtime *Runtime) Tracer() traceapi.Tracer { return runtime.tracer }
func (runtime *Runtime) Meter() metricapi.Meter  { return runtime.meter }
func (runtime *Runtime) Propagator() propagation.TextMapPropagator {
	return runtime.propagator
}

func (runtime *Runtime) Shutdown(ctx context.Context) error {
	if runtime == nil {
		return nil
	}
	return errors.Join(runtime.meterProvider.Shutdown(ctx), runtime.tracerProvider.Shutdown(ctx))
}
