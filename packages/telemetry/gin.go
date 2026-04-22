package telemetry

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("gin-server")

type httpHeaderCarrier map[string][]string

func (h httpHeaderCarrier) Get(key string) string {
	if v, ok := h[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

func (h httpHeaderCarrier) Set(key string, value string) {
	h[key] = []string{value}
}

func (h httpHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}

func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		h := httpHeaderCarrier(c.Request.Header)
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.TextMapCarrier(h))

		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithAttributes(
				attribute.String("http.request.method", c.Request.Method),
				attribute.String("url.path", c.Request.URL.Path),
				attribute.String("url.query", c.Request.URL.RawQuery),
			),
		)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		span.SetAttributes(
			attribute.Int("http.response.status_code", c.Writer.Status()),
		)
	}
}
