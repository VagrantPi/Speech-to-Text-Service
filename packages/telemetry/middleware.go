package telemetry

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

func MetricsMiddleware(meter metric.Meter) gin.HandlerFunc {
	counter, _ := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	histogram, _ := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
	)

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		path := c.FullPath()
		if path == "" {
			path = c.Request.Method + " " + c.Request.URL.Path
		}

		counter.Add(c.Request.Context(), 1,
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("path", path),
				attribute.Int("status_code", c.Writer.Status()),
			),
		)
		histogram.Record(c.Request.Context(), duration,
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("path", path),
			),
		)
	}
}

func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		status := c.Writer.Status()

		log := WithTraceID(c.Request.Context(), logger)
		log.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.Int("status", status),
			zap.Duration("latency_ms", duration),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}
