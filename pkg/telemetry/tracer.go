package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry configuration
type Config struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	CollectorAddr  string
	// Metric-specific configuration
	MetricInterval time.Duration // Interval for metric export (default: 15s)
	// Trace-specific configuration
	SampleRatio float64 // Sample ratio for traces (default: 1.0 = always sample)
}

// Telemetry holds the tracer provider, meter provider, tracer, and meter
type Telemetry struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter
	config         *Config
	resource       *resource.Resource
}

var globalTelemetry *Telemetry

// Init initializes OpenTelemetry with the given configuration
func Init(ctx context.Context, cfg *Config) (*Telemetry, error) {
	if cfg == nil || !cfg.Enabled {
		// Return a no-op tracer/meter if disabled
		serviceName := "unknown"
		if cfg != nil {
			serviceName = cfg.ServiceName
		}
		globalTelemetry = &Telemetry{
			tracer: otel.Tracer(serviceName),
			meter:  otel.Meter(serviceName),
			config: cfg,
		}
		return globalTelemetry, nil
	}

	// Apply defaults
	if cfg.MetricInterval == 0 {
		cfg.MetricInterval = 15 * time.Second
	}
	if cfg.SampleRatio == 0 {
		cfg.SampleRatio = 1.0 // Always sample by default
	}

	// Create resource with service information
	res, err := createResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create TracerProvider
	tracerProvider, err := createTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	// Create MeterProvider
	meterProvider, err := createMeterProvider(ctx, cfg, res)
	if err != nil {
		// Shutdown tracer provider if meter provider fails
		_ = tracerProvider.Shutdown(ctx)
		return nil, fmt.Errorf("failed to create meter provider: %w", err)
	}

	// Set global providers and propagator
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	globalTelemetry = &Telemetry{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		tracer:         tracerProvider.Tracer(cfg.ServiceName),
		meter:          meterProvider.Meter(cfg.ServiceName),
		config:         cfg,
		resource:       res,
	}

	return globalTelemetry, nil
}

// createResource creates a resource with service information and attributes
func createResource(cfg *Config) (*resource.Resource, error) {
	// Create service resource without merging with Default() to avoid schema URL conflicts
	// The default resource uses a newer schema URL that conflicts with semconv v1.27.0
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.DeploymentEnvironmentNameKey.String(cfg.Environment),
		attribute.String("service.namespace", "booking-rush"),
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKNameKey.String("opentelemetry"),
		semconv.TelemetrySDKVersionKey.String("1.39.0"),
	), nil
}

// createTracerProvider creates and configures the TracerProvider
func createTracerProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	// Create OTLP trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.CollectorAddr),
		otlptracegrpc.WithInsecure(), // Use insecure for internal network
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create sampler based on config
	var sampler sdktrace.Sampler
	if cfg.SampleRatio >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRatio <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRatio)
	}

	// Create TracerProvider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	return provider, nil
}

// createMeterProvider creates and configures the MeterProvider
func createMeterProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	// Create OTLP metric exporter
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.CollectorAddr),
		otlpmetricgrpc.WithInsecure(), // Use insecure for internal network
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	// Create MeterProvider with periodic reader
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExporter,
				sdkmetric.WithInterval(cfg.MetricInterval),
			),
		),
	)

	return provider, nil
}

// Shutdown gracefully shuts down both tracer and meter providers
func Shutdown(ctx context.Context) error {
	if globalTelemetry == nil {
		return nil
	}

	var errs []error

	// Shutdown tracer provider
	if globalTelemetry.tracerProvider != nil {
		if err := globalTelemetry.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer provider shutdown: %w", err))
		}
	}

	// Shutdown meter provider
	if globalTelemetry.meterProvider != nil {
		if err := globalTelemetry.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter provider shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// Get returns the global telemetry instance
func Get() *Telemetry {
	return globalTelemetry
}

// Tracer returns the tracer
func (t *Telemetry) Tracer() trace.Tracer {
	return t.tracer
}

// Meter returns the meter for creating metrics
func (t *Telemetry) Meter() metric.Meter {
	return t.meter
}

// Resource returns the resource
func (t *Telemetry) Resource() *resource.Resource {
	return t.resource
}

// Config returns the telemetry configuration
func (t *Telemetry) Config() *Config {
	return t.config
}

// GetMeter returns the global meter instance
func GetMeter() metric.Meter {
	if globalTelemetry == nil || globalTelemetry.meter == nil {
		return otel.Meter("noop")
	}
	return globalTelemetry.meter
}

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if globalTelemetry == nil || globalTelemetry.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return globalTelemetry.tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// GetTraceID returns the trace ID from context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().HasTraceID() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// GetSpanID returns the span ID from context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().HasSpanID() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanError records an error on the current span
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}
