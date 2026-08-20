// internal/webhook/security.go
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

func VerifyWebhookHandlerFiber(c *fiber.Ctx, cfg config.Config) (bool, error) {
	signatureHeader := c.Get("X-Hub-Signature-256", "")
	if signatureHeader == "" {
		return false, fmt.Errorf("assinatura X-Hub-Signature-256 não encontrada")
	}

	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return false, fmt.Errorf("formato de assinatura inválido")
	}
	signature := strings.TrimPrefix(signatureHeader, "sha256=")
	if signature == "" {
		return false, fmt.Errorf("formato de assinatura inválido")
	}

	secret := strings.TrimSpace(cfg.WhatsAppAppSecret)
	if secret == "" {
		// FAIL CLOSED em produção
		if cfg.IsProduction() {
			return false, fmt.Errorf("WHATSAPP_APP_SECRET não configurado")
		}
		logger.Warn(c.UserContext(), "WHATSAPP_APP_SECRET não configurado - validação desabilitada em dev")
		return true, nil
	}

	mac := hmac.New(sha256.New, []byte(secret))
	// Fiber: c.Body() retorna raw body, ok pra HMAC
	mac.Write(c.Body())
	expectedMAC := mac.Sum(nil)

	receivedMAC, err := hex.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("assinatura com hex inválido")
	}

	if !hmac.Equal(receivedMAC, expectedMAC) {
		logger.Warn(c.UserContext(), "Assinatura inválida do webhook",
			zap.String("remote_ip", c.IP()),
			zap.Int("body_len", len(c.Body())),
			// NÃO loga expected/received/secret
		)
		return false, fmt.Errorf("assinatura inválida")
	}

	return true, nil
}
