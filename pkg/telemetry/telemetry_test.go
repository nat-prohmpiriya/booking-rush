package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestInit_Disabled(t *testing.T) {
	ctx := context.Background()

	// Test with nil config
	tel, err := Init(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, tel)
	assert.NotNil(t, tel.tracer)
	assert.NotNil(t, tel.meter)

	// Test with disabled config
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
	}
	tel, err = Init(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, tel)
	assert.NotNil(t, tel.tracer)
	assert.NotNil(t, tel.meter)
	assert.Equal(t, cfg, tel.config)
}

func TestInit_Enabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := &Config{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		CollectorAddr:  "localhost:4317",
		MetricInterval: 10 * time.Second,
		SampleRatio:    1.0,
	}

	tel, err := Init(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, tel)
	assert.NotNil(t, tel.tracerProvider)
	assert.NotNil(t, tel.meterProvider)
	assert.NotNil(t, tel.tracer)
	assert.NotNil(t, tel.meter)
	assert.NotNil(t, tel.resource)

	// Verify global telemetry is set
	assert.Equal(t, tel, Get())

	// Cleanup
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()
	_ = Shutdown(shutdownCtx)
}

func TestInit_DefaultValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := &Config{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		CollectorAddr:  "localhost:4317",
		// Leave MetricInterval and SampleRatio as 0 to test defaults
	}

	tel, err := Init(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, tel)

	// Verify defaults are applied
	assert.Equal(t, 15*time.Second, cfg.MetricInterval)
	assert.Equal(t, 1.0, cfg.SampleRatio)

	// Cleanup
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()
	_ = Shutdown(shutdownCtx)
}

func TestShutdown_NilGlobal(t *testing.T) {
	globalTelemetry = nil
	err := Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestGet(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tel, err := Init(ctx, cfg)
	require.NoError(t, err)

	got := Get()
	assert.Equal(t, tel, got)
}

func TestTelemetry_Accessors_Disabled(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		Enabled:        false,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	tel, err := Init(ctx, cfg)
	require.NoError(t, err)

	// Test accessors for disabled telemetry
	assert.NotNil(t, tel.Tracer())
	assert.NotNil(t, tel.Meter())
	assert.Nil(t, tel.Resource()) // Resource is nil when disabled
	assert.Equal(t, cfg, tel.Config())
}

func TestStartSpan_Disabled(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		Enabled:        false,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	_, err := Init(ctx, cfg)
	require.NoError(t, err)

	// Test StartSpan with disabled telemetry (returns noop span)
	newCtx, span := StartSpan(ctx, "test-span")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, span)
	span.End()
}

func TestStartSpan_NilGlobal(t *testing.T) {
	globalTelemetry = nil
	ctx := context.Background()

	newCtx, span := StartSpan(ctx, "test-span")
	assert.Equal(t, ctx, newCtx)
	assert.NotNil(t, span)
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
	}

	_, err := Init(ctx, cfg)
	require.NoError(t, err)

	// SpanFromContext should work even with no active span
	span := SpanFromContext(ctx)
	assert.NotNil(t, span)
}

func TestGetTraceID_NoSpan(t *testing.T) {
	ctx := context.Background()

	// Test with no span
	traceID := GetTraceID(ctx)
	assert.Empty(t, traceID)
}

func TestGetSpanID_NoSpan(t *testing.T) {
	ctx := context.Background()

	// Test with no span
	spanID := GetSpanID(ctx)
	assert.Empty(t, spanID)
}

func TestAddSpanEvent_NoSpan(t *testing.T) {
	ctx := context.Background()

	// Should not panic even with no span
	AddSpanEvent(ctx, "test-event", attribute.String("key", "value"))
}

func TestSetSpanError_NoSpan(t *testing.T) {
	ctx := context.Background()

	// Should not panic even with no span
	SetSpanError(ctx, assert.AnError)
}

func TestSetSpanAttributes_NoSpan(t *testing.T) {
	ctx := context.Background()

	// Should not panic even with no span
	SetSpanAttributes(ctx, attribute.String("key", "value"), attribute.Int("number", 42))
}

func TestGetMeter_Disabled(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tel, err := Init(ctx, cfg)
	require.NoError(t, err)

	meter := GetMeter()
	assert.Equal(t, tel.meter, meter)
}

func TestGetMeter_NilGlobal(t *testing.T) {
	globalTelemetry = nil
	meter := GetMeter()
	assert.NotNil(t, meter) // Should return noop meter
}

func TestCreateResource(t *testing.T) {
	cfg := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	res, err := createResource(cfg)
	require.NoError(t, err)
	assert.NotNil(t, res)

	// Check that resource has expected attributes
	attrs := res.Attributes()
	assert.True(t, len(attrs) > 0)

	// Verify service name attribute is present
	foundServiceName := false
	for _, attr := range attrs {
		if string(attr.Key) == "service.name" {
			assert.Equal(t, "test-service", attr.Value.AsString())
			foundServiceName = true
			break
		}
	}
	assert.True(t, foundServiceName, "service.name attribute not found")
}

func TestConfig_Defaults(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		ServiceName: "test",
	}

	// These should be zero initially
	assert.Equal(t, time.Duration(0), cfg.MetricInterval)
	assert.Equal(t, 0.0, cfg.SampleRatio)
}
