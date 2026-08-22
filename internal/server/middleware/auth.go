package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

func AuthMiddlewareFiber(authService service.AuthServiceInterface) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{"error": "token required"})
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := authService.ValidateToken(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "token invalid"})
		}
		c.Locals("userID", claims.UserID)
		c.Locals("tenantID", claims.TenantID)
		c.Locals("role", claims.Role)
		return c.Next()
	}
}
