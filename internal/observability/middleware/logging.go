// internal/observability/middleware/logging.go
package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
)

// LoggingMiddleware cria um middleware para logging de requisições HTTP
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Gera trace_id se não existir
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = logger.GenerateTraceID()
		}

		// Cria logger com trace_id
		ctx := logger.WithTraceID(r.Context(), traceID)
		ctx = logger.WithFields(ctx, zap.String("path", r.URL.Path))

		// Adiciona ao request
		r = r.WithContext(ctx)

		// Log da requisição
		logger.Info(ctx, "Request received",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)

		// Wrapper para capturar status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Processa a requisição
		next.ServeHTTP(wrapped, r)

		// Log da resposta
		duration := time.Since(start)
		logger.Info(ctx, "Request completed",
			zap.Int("status", wrapped.statusCode),
			zap.Duration("duration", duration),
			zap.Int64("duration_ms", duration.Milliseconds()),
		)
	})
}

// responseWriter wrapper para capturar o status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
