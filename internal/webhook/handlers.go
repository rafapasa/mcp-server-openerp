// internal/webhook/handlers.go - só HTTP, sem regra de negócio
package webhook

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	whatsappdto "github.com/rafapasa/mcp-server-openerp/internal/dto/whatsapp"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	cfg       *config.Config
	processor *Processor
}

func NewWebhookHandler(cfg *config.Config, processor *Processor) *WebhookHandler {
	return &WebhookHandler{cfg: cfg, processor: processor}
}

func (h *WebhookHandler) HandleWebhookFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	if c.Method() == fiber.MethodGet {
		mode := c.Query("hub.mode")
		token := c.Query("hub.verify_token")
		challenge := c.Query("hub.challenge")
		if mode == "subscribe" && token == h.cfg.WhatsAppVerifyToken {
			logger.Info(ctx, "Verificação de segurança do webhook realizada com sucesso")
			return c.Status(200).SendString(challenge)
		}
		logger.Warn(ctx, "Falha na verificação de segurança do webhook")
		return c.Status(403).SendString("forbidden")
	}

	ok, err := VerifyWebhookHandlerFiber(c, *h.cfg)
	if !ok {
		logger.Warn(ctx, "HMAC falhou", zap.Error(err), zap.String("ip", c.IP()))
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	body := c.Body()
	if len(body) == 0 {
		return c.Status(200).JSON(fiber.Map{"success": true})
	}

	if !strings.Contains(string(body), "\"messages\"") {
		return c.Status(200).JSON(fiber.Map{"success": true})
	}

	var req whatsappdto.WebhookRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error(ctx, "JSON inválido", zap.Error(err))
		return c.Status(200).JSON(fiber.Map{"success": true})
	}

	detachedCtx := context.WithoutCancel(ctx)
	go h.processor.Process(detachedCtx, req)

	return c.Status(200).JSON(fiber.Map{"success": true})
}
