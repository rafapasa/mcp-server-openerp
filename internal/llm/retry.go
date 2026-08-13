// internal/llm/retry.go
package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

// RetryConfig configuração de retry
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Backoff     func(attempt int) time.Duration
}

// DefaultRetryConfig retorna a configuração padrão
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    10 * time.Second,
		Backoff: func(attempt int) time.Duration {
			delay := time.Duration(1<<uint(attempt)) * time.Second
			if delay > 10*time.Second {
				delay = 10 * time.Second
			}
			return delay
		},
	}
}

// RetryWithBackoff executa uma função com retry e backoff exponencial
func RetryWithBackoff[T any](
	ctx context.Context,
	config RetryConfig,
	fn func() (T, error),
) (T, error) {
	var lastErr error
	var zero T

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Verifica se o contexto foi cancelado
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		result, err := fn()
		if err == nil {
			if attempt > 1 {
				logger.Info(ctx, "LLM chamada bem-sucedida após retry",
					zap.Int("attempt", attempt),
				)
			}
			return result, nil
		}

		lastErr = err

		// Verifica se o erro merece retry
		if !shouldRetry(err) {
			logger.Warn(ctx, "Erro não recuperável, abortando retry",
				zap.Error(err),
				zap.Int("attempt", attempt),
			)
			return zero, err
		}

		// Se não for a última tentativa, espera
		if attempt < config.MaxAttempts {
			delay := config.Backoff(attempt)
			logger.Warn(ctx, "Erro na LLM, tentando novamente",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.Int("max_attempts", config.MaxAttempts),
				zap.Duration("delay", delay),
			)

			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return zero, ctx.Err()
			}
		}
	}

	return zero, fmt.Errorf("todas as %d tentativas falharam: %w", config.MaxAttempts, lastErr)
}

// shouldRetry verifica se o erro merece retry
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Erros que merecem retry
	retryablePatterns := []string{
		"503",         // Service Unavailable
		"unavailable", // Indisponível
		"rate limit",  // Rate limiting
		"too many",    // Too many requests
		"429",         // Too Many Requests
		"500",         // Internal Server Error
		"502",         // Bad Gateway
		"504",         // Gateway Timeout
		"timeout",     // Timeout
		"deadline",    // Deadline exceeded
		"temporary",   // Erro temporário
		"high demand", // Alta demanda (como o erro do Gemini)
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}
