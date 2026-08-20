// logging.go - Fiber
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

func LoggerFiber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		// log entrada
		logger.GetLogger().Info("➡️  REQ",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.String("query", string(c.Request().RequestURI())),
		)

		err := c.Next()

		logger.GetLogger().Info("⬅️  RES",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", time.Since(start)),
		)
		return err
	}
}
