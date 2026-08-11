// internal/observability/middleware/tracing.go
package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/tracing"
)

// TracingMiddleware cria um middleware para tracing HTTP
func TracingMiddleware(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "http.server",
		otelhttp.WithTracerProvider(tracing.GetTracerProvider()), // Use the specific TracerProvider
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),    // Use the global TextMapPropagator
	)
}

// HTTPTraceWrapper é um wrapper para tracing de requisições HTTP
func HTTPTraceWrapper(handler http.Handler) http.Handler {
	return TracingMiddleware(handler)
}
