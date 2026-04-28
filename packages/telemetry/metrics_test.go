package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetrick "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.uber.org/zap"
)

func setupTestMeter(t *testing.T) (*sdkmetrick.MeterProvider, func()) {
	reader := sdkmetrick.NewManualReader()
	res, err := resource.New(context.Background(),
		resource.WithAttributes(semconv.ServiceName("test")),
	)
	require.NoError(t, err)

	mp := sdkmetrick.NewMeterProvider(
		sdkmetrick.WithReader(reader),
		sdkmetrick.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	return mp, func() {
		mp.Shutdown(context.Background())
	}
}

func TestNewCounter(t *testing.T) {
	_, cleanup := setupTestMeter(t)
	defer cleanup()

	counter, err := NewCounter("test_counter", "test description")
	assert.NoError(t, err)
	assert.NotNil(t, counter)

	counter.Add(context.Background(), 1, metric.WithAttributes(attribute.String("status", "ok")))
}

func TestNewHistogram(t *testing.T) {
	_, cleanup := setupTestMeter(t)
	defer cleanup()

	histogram, err := NewHistogram("test_histogram", "test description", []float64{0.1, 0.5, 1.0})
	assert.NoError(t, err)
	assert.NotNil(t, histogram)

	histogram.Record(context.Background(), 0.3)
}

func TestNewLogger(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		logger, err := NewLogger("test-service")
		assert.NoError(t, err)
		assert.NotNil(t, logger)
		logger.Sync()
	})

	t.Run("with context - no trace id", func(t *testing.T) {
		logger, _ := zap.NewDevelopment()
		result := WithTraceID(context.Background(), logger)
		assert.NotNil(t, result)
	})

	t.Run("with valid context", func(t *testing.T) {
		_, cleanup := setupTestMeter(t)
		defer cleanup()

		tracer := otel.Tracer("test")
		ctx, _ := tracer.Start(context.Background(), "test-span")
		logger, _ := zap.NewDevelopment()
		result := WithTraceID(ctx, logger)
		assert.NotNil(t, result)
		logger.Sync()
	})
}