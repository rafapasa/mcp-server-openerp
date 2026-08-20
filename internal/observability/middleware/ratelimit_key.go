package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// KeyExtractor extrai a chave de rate limit no Fiber
type KeyExtractor func(c *fiber.Ctx) string

// TenantKeyExtractor - prioridade: tenant_id > IP
func TenantKeyExtractor(c *fiber.Ctx) string {
	// Header X-Tenant-ID (seu multi-tenant)
	if tenant := c.Get("X-Tenant-ID"); tenant != "" {
		return "tenant:" + strings.TrimSpace(tenant)
	}
	// Query?tenant_id=
	if tenant := c.Query("tenant_id"); tenant != "" {
		return "tenant:" + strings.TrimSpace(tenant)
	}
	// Fallback IP
	return "ip:" + c.IP()
}

// ClientKeyExtractor - tenant + cliente (mais granular pro webhook)
func ClientKeyExtractor(c *fiber.Ctx) string {
	tenant := c.Get("X-Tenant-ID")
	client := c.Get("X-Client-ID")

	if tenant != "" && client != "" {
		return "client:" + tenant + ":" + client
	}
	if tenant != "" {
		return "tenant:" + tenant
	}
	return "ip:" + c.IP()
}

// IPKeyExtractor - só IP
func IPKeyExtractor(c *fiber.Ctx) string {
	return "ip:" + c.IP()
}

// WebhookKeyExtractor - ESPECÍFICO pro seu caso
// Usa o telefone do cliente (from) se tiver, senão IP
func WebhookKeyExtractor(c *fiber.Ctx) string {
	// Se quiser rate limit por número de telefone do WhatsApp,
	// você pode parsear o body depois, mas pro middleware rápido,
	// usa tenant ou IP
	if tenant := c.Get("X-Tenant-ID"); tenant != "" {
		return "tenant:" + tenant
	}
	return "ip:" + c.IP()
}
