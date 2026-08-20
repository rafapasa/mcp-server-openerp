package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/metrics"
)

func MetricsFiber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		// Não mede health/metrics pra não poluir
		if c.Path() == "/metrics" {
			return err
		}

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())
		path := c.Route().Path // usa rota registrada /webhook, não /webhook?hub.mode=...
		if path == "" {
			path = c.Path()
		}

		if metrics.HttpRequestsTotal != nil {
			metrics.HttpRequestsTotal.WithLabelValues(c.Method(), path, status).Inc()
		}
		if metrics.HttpRequestDuration != nil {
			metrics.HttpRequestDuration.WithLabelValues(c.Method(), path).Observe(duration)
		}

		return err
	}
}
