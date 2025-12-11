package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricOpts holds options for creating metrics
type MetricOpts struct {
	Name        string
	Description string
	Unit        string
}

// Counter wraps an OTel counter for easier use
type Counter struct {
	counter metric.Int64Counter
}

// NewCounter creates a new counter metric
func NewCounter(opts MetricOpts) (*Counter, error) {
	meter := GetMeter()
	counter, err := meter.Int64Counter(
		opts.Name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
	)
	if err != nil {
		return nil, err
	}
	return &Counter{counter: counter}, nil
}

// Add increments the counter by the given value
func (c *Counter) Add(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

// Inc increments the counter by 1
func (c *Counter) Inc(ctx context.Context, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Gauge wraps an OTel gauge for easier use
type Gauge struct {
	gauge    metric.Int64Gauge
	callback metric.Registration
}

// NewGauge creates a new gauge metric
func NewGauge(opts MetricOpts) (*Gauge, error) {
	meter := GetMeter()
	gauge, err := meter.Int64Gauge(
		opts.Name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
	)
	if err != nil {
		return nil, err
	}
	return &Gauge{gauge: gauge}, nil
}

// Record sets the gauge to the given value
func (g *Gauge) Record(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	g.gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}

// NewGaugeWithCallback creates a gauge that calls the provided function to get its value
func NewGaugeWithCallback(opts MetricOpts, callback func() int64, attrs ...attribute.KeyValue) (*Gauge, error) {
	meter := GetMeter()
	gauge := &Gauge{}

	registration, err := meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			value := callback()
			asyncGauge, err := meter.Int64ObservableGauge(
				opts.Name,
				metric.WithDescription(opts.Description),
				metric.WithUnit(opts.Unit),
			)
			if err != nil {
				return err
			}
			observer.ObserveInt64(asyncGauge, value, metric.WithAttributes(attrs...))
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	gauge.callback = registration
	return gauge, nil
}

// Histogram wraps an OTel histogram for easier use
type Histogram struct {
	histogram metric.Float64Histogram
}

// NewHistogram creates a new histogram metric
func NewHistogram(opts MetricOpts) (*Histogram, error) {
	meter := GetMeter()
	histogram, err := meter.Float64Histogram(
		opts.Name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
	)
	if err != nil {
		return nil, err
	}
	return &Histogram{histogram: histogram}, nil
}

// Record records a value in the histogram
func (h *Histogram) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	h.histogram.Record(ctx, value, metric.WithAttributes(attrs...))
}

// NewHistogramWithBuckets creates a new histogram with custom bucket boundaries
func NewHistogramWithBuckets(opts MetricOpts, boundaries []float64) (*Histogram, error) {
	meter := GetMeter()
	histogram, err := meter.Float64Histogram(
		opts.Name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
		metric.WithExplicitBucketBoundaries(boundaries...),
	)
	if err != nil {
		return nil, err
	}
	return &Histogram{histogram: histogram}, nil
}

// UpDownCounter wraps an OTel up-down counter for values that can increase and decrease
type UpDownCounter struct {
	counter metric.Int64UpDownCounter
}

// NewUpDownCounter creates a new up-down counter metric
func NewUpDownCounter(opts MetricOpts) (*UpDownCounter, error) {
	meter := GetMeter()
	counter, err := meter.Int64UpDownCounter(
		opts.Name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
	)
	if err != nil {
		return nil, err
	}
	return &UpDownCounter{counter: counter}, nil
}

// Add adds the given value to the counter (can be negative)
func (c *UpDownCounter) Add(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

// Inc increments the counter by 1
func (c *UpDownCounter) Inc(ctx context.Context, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Dec decrements the counter by 1
func (c *UpDownCounter) Dec(ctx context.Context, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, -1, metric.WithAttributes(attrs...))
}

// Common metric attribute keys
const (
	AttrServiceName   = "service.name"
	AttrEnvironment   = "environment"
	AttrMethod        = "http.method"
	AttrPath          = "http.path"
	AttrStatusCode    = "http.status_code"
	AttrErrorType     = "error.type"
	AttrEventID       = "event.id"
	AttrUserID        = "user.id"
	AttrTenantID      = "tenant.id"
	AttrBookingStatus = "booking.status"
	AttrPaymentStatus = "payment.status"
)

// Helper functions for common attributes
func ServiceAttr(name string) attribute.KeyValue {
	return attribute.String(AttrServiceName, name)
}

func EnvironmentAttr(env string) attribute.KeyValue {
	return attribute.String(AttrEnvironment, env)
}

func MethodAttr(method string) attribute.KeyValue {
	return attribute.String(AttrMethod, method)
}

func PathAttr(path string) attribute.KeyValue {
	return attribute.String(AttrPath, path)
}

func StatusCodeAttr(code int) attribute.KeyValue {
	return attribute.Int(AttrStatusCode, code)
}

func ErrorTypeAttr(errType string) attribute.KeyValue {
	return attribute.String(AttrErrorType, errType)
}

func EventIDAttr(eventID string) attribute.KeyValue {
	return attribute.String(AttrEventID, eventID)
}

func UserIDAttr(userID string) attribute.KeyValue {
	return attribute.String(AttrUserID, userID)
}

func TenantIDAttr(tenantID string) attribute.KeyValue {
	return attribute.String(AttrTenantID, tenantID)
}

func BookingStatusAttr(status string) attribute.KeyValue {
	return attribute.String(AttrBookingStatus, status)
}

func PaymentStatusAttr(status string) attribute.KeyValue {
	return attribute.String(AttrPaymentStatus, status)
}
