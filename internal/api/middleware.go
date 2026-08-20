// internal/api/middleware.go - FINAL 100% FIBER - SEM net/http
package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	TenantIDKey contextKey = "tenant_id"
	EmailKey    contextKey = "email"
)

// AuthMiddlewareFiber - verifica JWT no Fiber
func AuthMiddlewareFiber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"success": false,
				"error":   "Authorization header required",
			})
		}

		tokenString := ExtractTokenFromHeader(authHeader)
		if tokenString == "" {
			return c.Status(401).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid authorization header format",
			})
		}

		claims, err := ValidateJWT(tokenString)
		if err != nil {
			logger.Warn(c.UserContext(), "Invalid JWT", zap.Error(err))
			return c.Status(401).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid or expired token",
			})
		}

		// Fiber usa Locals em vez de context.WithValue
		c.Locals(string(UserIDKey), claims.UserID)
		c.Locals(string(TenantIDKey), claims.TenantID)
		c.Locals(string(EmailKey), claims.Email)
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)

		return c.Next()
	}
}

// OptionalAuthMiddlewareFiber - auth opcional
func OptionalAuthMiddlewareFiber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			tokenString := ExtractTokenFromHeader(authHeader)
			if tokenString != "" {
				claims, err := ValidateJWT(tokenString)
				if err == nil {
					c.Locals(string(UserIDKey), claims.UserID)
					c.Locals(string(TenantIDKey), claims.TenantID)
					c.Locals(string(EmailKey), claims.Email)
					c.Locals("tenant_id", claims.TenantID)
				}
			}
		}
		return c.Next()
	}
}

// Helpers Fiber
func GetUserIDFiber(c *fiber.Ctx) (uint, bool) {
	if v := c.Locals("user_id"); v != nil {
		if id, ok := v.(uint); ok {
			return id, true
		}
	}
	if v := c.Locals(string(UserIDKey)); v != nil {
		if id, ok := v.(uint); ok {
			return id, true
		}
	}
	return 0, false
}

func GetTenantIDFiber(c *fiber.Ctx) (uint, bool) {
	// tenta Locals primeiro (setado pelo AuthMiddlewareFiber)
	if v := c.Locals("tenant_id"); v != nil {
		switch val := v.(type) {
		case uint:
			return val, true
		case int:
			return uint(val), true
		case int64:
			return uint(val), true
		}
	}
	if v := c.Locals(string(TenantIDKey)); v != nil {
		if id, ok := v.(uint); ok {
			return id, true
		}
	}
	// fallback header (dev)
	if tid := c.Get("X-Tenant-ID"); tid != "" {
		if id, err := parseUintFiber(tid); err == nil && id != 0 {
			return id, true
		}
	}
	return 0, false
}

func GetTenantID(c *fiber.Ctx) (uint, bool) {
	return GetTenantIDFiber(c)
}

// Compatibilidade - mantém assinatura antiga pra não quebrar, mas redireciona pro Fiber
// Pode deletar depois que tudo for Fiber
func ExtractTokenFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}
	return ""
}
