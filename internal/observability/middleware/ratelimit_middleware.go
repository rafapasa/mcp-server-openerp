// internal/observability/middleware/ratelimit_middleware.go
package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

// RateLimitMiddleware cria um middleware para rate limiting
func RateLimitMiddleware(rl *RateLimiter, extractor KeyExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extrai a chave
			key := extractor(r)

			// Verifica se a requisição é permitida
			allowed, retryAfter := rl.Allow(key)

			if !allowed {
				// Log do bloqueio
				logger.Warn(r.Context(), "Rate limit excedido",
					zap.String("key", key),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
					zap.Int64("retry_after_seconds", retryAfter),
				)

				// Headers de rate limit
				w.Header().Set("Retry-After", formatRetryAfter(retryAfter))
				w.Header().Set("X-RateLimit-Limit", "Excedido")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", time.Now().Add(time.Duration(retryAfter)*time.Second).Format(time.RFC3339))

				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			// Adiciona headers informativos (opcional)
			// w.Header().Set("X-RateLimit-Remaining", "implementar")

			next.ServeHTTP(w, r)
		})
	}
}

// formatRetryAfter formata o tempo de espera
func formatRetryAfter(seconds int64) string {
	if seconds < 60 {
		return strconv.FormatInt(seconds, 10)
	}
	return time.Now().Add(time.Duration(seconds) * time.Second).Format(time.RFC1123)
}
