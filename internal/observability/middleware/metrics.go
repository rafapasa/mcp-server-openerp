// internal/observability/middleware/metrics.go
package middleware

import (
	"net/http"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/metrics"
)

// MetricsMiddleware cria um middleware para coleta de métricas HTTP
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrapper para capturar status code e tamanho
		wrapped := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Processa a requisição
		next.ServeHTTP(wrapped, r)

		// Coleta métricas
		duration := time.Since(start).Seconds()
		path := r.URL.Path
		method := r.Method
		status := http.StatusText(wrapped.statusCode)

		metrics.RequestTotal.WithLabelValues(method, path, status).Inc()
		metrics.RequestDuration.WithLabelValues(method, path).Observe(duration)
		metrics.RequestSize.WithLabelValues(method, path).Observe(float64(r.ContentLength))
		metrics.ResponseSize.WithLabelValues(method, path).Observe(float64(wrapped.size))
	})
}

// metricsResponseWriter wrapper para capturar status code e tamanho
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *metricsResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}
