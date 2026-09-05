// internal/webhook/security.go
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	whatsappdto "github.com/rafapasa/mcp-server-openerp/internal/dto/whatsapp"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/pkg/phone"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

// VerifyWebhookHandlerFiber valida a segurança do webhook:
//  1. Localiza o tenant pelo phone_number_id (fallback display_phone_number) do payload,
//     usando as informações da tabela `tenants` via service (cache Redis 1h).
//  2. Valida a assinatura HMAC X-Hub-Signature-256 com o app secret do Meta App.
//
// O app secret é global (único por Meta App), por isso continua vindo do .env
// (WHATSAPP_APP_SECRET) — não existe por tenant. O verify_token, esse sim por tenant,
// é validado no GET /webhook (handlers.go) contra a coluna whatsapp_verify_token.
func VerifyWebhookHandlerFiber(c *fiber.Ctx, cfg config.Config, tenantService service.TenantServiceInterface) (bool, error) {
	// 1. Localiza o tenant (banco + cache) - mesma lógica do processor
	tenant, err := tenantPeloPayload(c, tenantService)
	if err != nil || tenant == nil {
		return false, fmt.Errorf("tenant não localizado para o webhook: %w", err)
	}

	// 2. Valida assinatura HMAC (app secret global)
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
		return false, fmt.Errorf("WHATSAPP_APP_SECRET não configurado")
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
		logger.Warn(
			c.UserContext(), "Assinatura inválida do webhook",
			zap.String("remote_ip", c.IP()),
			zap.Int("body_len", len(c.Body())),
			// NÃO loga expected/received/secret
		)
		return false, fmt.Errorf("assinatura inválida")
	}

	return true, nil
}

// tenantPeloPayload resolve o tenant a partir do phone_number_id do payload da Meta
// (fallback para display_phone_number), replicando o critério do processor.
func tenantPeloPayload(c *fiber.Ctx, tenantService service.TenantServiceInterface) (*dto.TenantDTO, error) {
	if tenantService == nil {
		return nil, fmt.Errorf("tenantService não injetado")
	}

	body := c.Body()
	if len(body) == 0 {
		return nil, fmt.Errorf("body vazio")
	}

	var req whatsappdto.WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("parse do payload: %w", err)
	}

	for _, entry := range req.Entry {
		for _, change := range entry.Changes {
			md := change.Value.Metadata
			// 1. phone_number_id (definitivo)
			if md.PhoneNumberID != "" {
				if t, err := tenantService.GetByWhatsAppPhoneID(c.UserContext(), md.PhoneNumberID); err == nil && t != nil && t.ID != 0 {
					return t, nil
				}
			}
			// 2. fallback display_phone_number
			if clean := phone.Normalize(md.DisplayPhoneNumber); clean != "" {
				if t, err := tenantService.GetByTelefone(c.UserContext(), clean); err == nil && t != nil && t.ID != 0 {
					return t, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("nenhum tenant encontrado para o payload")
}
