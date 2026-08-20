package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
)

func SecurityHeadersFiber(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// HSTS só em prod
		if cfg.IsProduction() {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=()")
		c.Set("Content-Security-Policy", buildCSPFiber())

		c.Response().Header.Del("Server")
		c.Response().Header.Del("X-Powered-By")

		return c.Next()
	}
}

func buildCSPFiber() string {
	directives := []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: https:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"frame-src 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}
	return strings.Join(directives, "; ")
}
