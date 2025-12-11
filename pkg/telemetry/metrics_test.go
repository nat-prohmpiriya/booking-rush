package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func setupTelemetryForMetrics(t *testing.T) func() {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	cfg := &Config{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		CollectorAddr:  "localhost:4317",
	}

	_, err := Init(ctx, cfg)
	require.NoError(t, err)
	cancel()

	return func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer shutdownCancel()
		_ = Shutdown(shutdownCtx)
	}
}

func setupTelemetryDisabled(t *testing.T) func() {
	ctx := context.Background()
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
	}

	_, err := Init(ctx, cfg)
	require.NoError(t, err)

	return func() {
		_ = Shutdown(ctx)
	}
}

func TestNewCounter_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	counter, err := NewCounter(MetricOpts{
		Name:        "test_counter",
		Description: "A test counter",
		Unit:        "1",
	})
	require.NoError(t, err)
	assert.NotNil(t, counter)
}

func TestCounter_Add_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	counter, err := NewCounter(MetricOpts{
		Name:        "test_counter_add",
		Description: "A test counter for Add",
		Unit:        "1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Should not panic
	counter.Add(ctx, 5)
	counter.Add(ctx, 10, attribute.String("key", "value"))
}

func TestCounter_Inc_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	counter, err := NewCounter(MetricOpts{
		Name:        "test_counter_inc",
		Description: "A test counter for Inc",
		Unit:        "1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Should not panic
	counter.Inc(ctx)
	counter.Inc(ctx, attribute.String("key", "value"))
}

func TestNewGauge_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	gauge, err := NewGauge(MetricOpts{
		Name:        "test_gauge",
		Description: "A test gauge",
		Unit:        "1",
	})
	require.NoError(t, err)
	assert.NotNil(t, gauge)
}

func TestGauge_Record_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	gauge, err := NewGauge(MetricOpts{
		Name:        "test_gauge_record",
		Description: "A test gauge for Record",
		Unit:        "1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Should not panic
	gauge.Record(ctx, 42)
	gauge.Record(ctx, 100, attribute.String("key", "value"))
}

func TestNewHistogram_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	histogram, err := NewHistogram(MetricOpts{
		Name:        "test_histogram",
		Description: "A test histogram",
		Unit:        "ms",
	})
	require.NoError(t, err)
	assert.NotNil(t, histogram)
}

func TestHistogram_Record_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	histogram, err := NewHistogram(MetricOpts{
		Name:        "test_histogram_record",
		Description: "A test histogram for Record",
		Unit:        "ms",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Should not panic
	histogram.Record(ctx, 123.45)
	histogram.Record(ctx, 67.89, attribute.String("key", "value"))
}

func TestNewHistogramWithBuckets_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	boundaries := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	histogram, err := NewHistogramWithBuckets(MetricOpts{
		Name:        "test_histogram_buckets",
		Description: "A test histogram with custom buckets",
		Unit:        "s",
	}, boundaries)
	require.NoError(t, err)
	assert.NotNil(t, histogram)

	ctx := context.Background()

	// Should not panic
	histogram.Record(ctx, 0.1)
	histogram.Record(ctx, 1.5, attribute.String("key", "value"))
}

func TestNewUpDownCounter_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	counter, err := NewUpDownCounter(MetricOpts{
		Name:        "test_updown_counter",
		Description: "A test up-down counter",
		Unit:        "1",
	})
	require.NoError(t, err)
	assert.NotNil(t, counter)
}

func TestUpDownCounter_Add_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	counter, err := NewUpDownCounter(MetricOpts{
		Name:        "test_updown_counter_add",
		Description: "A test up-down counter for Add",
		Unit:        "1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Should not panic
	counter.Add(ctx, 5)
	counter.Add(ctx, -3)
	counter.Add(ctx, 10, attribute.String("key", "value"))
}

func TestUpDownCounter_IncDec_Disabled(t *testing.T) {
	cleanup := setupTelemetryDisabled(t)
	defer cleanup()

	counter, err := NewUpDownCounter(MetricOpts{
		Name:        "test_updown_counter_incdec",
		Description: "A test up-down counter for Inc/Dec",
		Unit:        "1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Should not panic
	counter.Inc(ctx)
	counter.Dec(ctx)
	counter.Inc(ctx, attribute.String("key", "value"))
	counter.Dec(ctx, attribute.String("key", "value"))
}

func TestAttributeHelpers(t *testing.T) {
	tests := []struct {
		name     string
		attrFunc func() attribute.KeyValue
		expected attribute.KeyValue
	}{
		{
			name: "ServiceAttr",
			attrFunc: func() attribute.KeyValue {
				return ServiceAttr("my-service")
			},
			expected: attribute.String(AttrServiceName, "my-service"),
		},
		{
			name: "EnvironmentAttr",
			attrFunc: func() attribute.KeyValue {
				return EnvironmentAttr("production")
			},
			expected: attribute.String(AttrEnvironment, "production"),
		},
		{
			name: "MethodAttr",
			attrFunc: func() attribute.KeyValue {
				return MethodAttr("GET")
			},
			expected: attribute.String(AttrMethod, "GET"),
		},
		{
			name: "PathAttr",
			attrFunc: func() attribute.KeyValue {
				return PathAttr("/api/v1/users")
			},
			expected: attribute.String(AttrPath, "/api/v1/users"),
		},
		{
			name: "StatusCodeAttr",
			attrFunc: func() attribute.KeyValue {
				return StatusCodeAttr(200)
			},
			expected: attribute.Int(AttrStatusCode, 200),
		},
		{
			name: "ErrorTypeAttr",
			attrFunc: func() attribute.KeyValue {
				return ErrorTypeAttr("validation_error")
			},
			expected: attribute.String(AttrErrorType, "validation_error"),
		},
		{
			name: "EventIDAttr",
			attrFunc: func() attribute.KeyValue {
				return EventIDAttr("evt_123")
			},
			expected: attribute.String(AttrEventID, "evt_123"),
		},
		{
			name: "UserIDAttr",
			attrFunc: func() attribute.KeyValue {
				return UserIDAttr("user_456")
			},
			expected: attribute.String(AttrUserID, "user_456"),
		},
		{
			name: "TenantIDAttr",
			attrFunc: func() attribute.KeyValue {
				return TenantIDAttr("tenant_789")
			},
			expected: attribute.String(AttrTenantID, "tenant_789"),
		},
		{
			name: "BookingStatusAttr",
			attrFunc: func() attribute.KeyValue {
				return BookingStatusAttr("confirmed")
			},
			expected: attribute.String(AttrBookingStatus, "confirmed"),
		},
		{
			name: "PaymentStatusAttr",
			attrFunc: func() attribute.KeyValue {
				return PaymentStatusAttr("paid")
			},
			expected: attribute.String(AttrPaymentStatus, "paid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.attrFunc()
			assert.Equal(t, tt.expected.Key, got.Key)
			assert.Equal(t, tt.expected.Value, got.Value)
		})
	}
}

func TestMetricConstants(t *testing.T) {
	assert.Equal(t, "service.name", AttrServiceName)
	assert.Equal(t, "environment", AttrEnvironment)
	assert.Equal(t, "http.method", AttrMethod)
	assert.Equal(t, "http.path", AttrPath)
	assert.Equal(t, "http.status_code", AttrStatusCode)
	assert.Equal(t, "error.type", AttrErrorType)
	assert.Equal(t, "event.id", AttrEventID)
	assert.Equal(t, "user.id", AttrUserID)
	assert.Equal(t, "tenant.id", AttrTenantID)
	assert.Equal(t, "booking.status", AttrBookingStatus)
	assert.Equal(t, "payment.status", AttrPaymentStatus)
}
