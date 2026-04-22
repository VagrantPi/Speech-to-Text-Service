package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func NewLogger(serviceName string) (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	logger = logger.With(zap.String("service", serviceName))
	return logger, nil
}

func WithTraceID(ctx context.Context, logger *zap.Logger) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return logger
	}
	return logger.With(
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	)
}
