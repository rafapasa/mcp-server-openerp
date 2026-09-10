package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/server/response"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

func AuthMiddlewareFiber(authService service.AuthServiceInterface) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return response.Unauthorized(c, "token required")
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := authService.ValidateToken(token)
		if err != nil {
			return response.FromError(c, err)
		}
		c.Locals("userID", claims.UserID)
		c.Locals("tenantID", claims.TenantID)
		c.Locals("role", claims.Role)
		return c.Next()
	}
}
