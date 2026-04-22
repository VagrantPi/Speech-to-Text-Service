package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

type Config struct {
	ServiceName  string `mapstructure:"OTEL_SERVICE_NAME"`
	OTelEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
}

func Init(cfg Config) (func(context.Context) error, error) {
	shutdownTracer, err := InitTracer(cfg)
	if err != nil {
		return nil, err
	}

	shutdownMeter, err := InitMeter(cfg)
	if err != nil {
		shutdownTracer(context.Background())
		return nil, err
	}

	return func(ctx context.Context) error {
		if err := shutdownMeter(ctx); err != nil {
			return err
		}
		return shutdownTracer(ctx)
	}, nil
}

func InitTracer(cfg Config) (func(context.Context) error, error) {
	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTelEndpoint),
		otlptracegrpc.WithInsecure(),
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

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
