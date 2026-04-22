package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

func InitMeter(cfg Config) (func(context.Context) error, error) {
	ctx := context.Background()

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTelEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}

func NewCounter(name, description string) (metric.Int64Counter, error) {
	meter := otel.Meter(name)
	return meter.Int64Counter(name, metric.WithDescription(description))
}

func NewHistogram(name, description string, boundaries []float64) (metric.Float64Histogram, error) {
	meter := otel.Meter(name)
	return meter.Float64Histogram(name,
		metric.WithDescription(description),
		metric.WithExplicitBucketBoundaries(boundaries...),
	)
}
