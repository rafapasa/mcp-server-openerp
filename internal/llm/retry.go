package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    10 * time.Second,
	}
}

func (c RetryConfig) Backoff(attempt int) time.Duration {
	// exponencial: 1s, 2s, 4s... limitado por MaxDelay
	delay := c.BaseDelay * time.Duration(1<<uint(attempt-1))
	if delay > c.MaxDelay {
		delay = c.MaxDelay
	}
	return delay
}

func RetryWithBackoff[T any](
	ctx context.Context,
	cfg RetryConfig,
	fn func() (T, error),
) (T, error) {
	var lastErr error
	var zero T

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		result, err := fn()
		if err == nil {
			if attempt > 1 {
				logger.Info(ctx, "LLM ok após retry", zap.Int("attempt", attempt))
			}
			return result, nil
		}

		lastErr = err

		if !shouldRetry(err) {
			logger.Warn(ctx, "Erro não retryable", zap.Error(err))
			return zero, err
		}

		if attempt < cfg.MaxAttempts {
			delay := cfg.Backoff(attempt)
			logger.Warn(
				ctx, "LLM falhou, retry",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
			)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return zero, ctx.Err()
			}
		}
	}

	return zero, fmt.Errorf("falhou após %d tentativas: %w", cfg.MaxAttempts, lastErr)
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	patterns := []string{
		"429", "503", "500", "502", "504",
		"unavailable", "rate limit", "too many",
		"timeout", "deadline", "temporary",
		"overloaded", "resource exhausted", "high demand", "try again",
	}
	for _, p := range patterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}
