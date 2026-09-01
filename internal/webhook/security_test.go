// internal/webhook/security_test.go
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func assinaturaHMAC(t *testing.T, body []byte, secret string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookHandlerFiber_HMACValidoComTenant(t *testing.T) {
	body := `{"entry":[{"changes":[{"value":{"metadata":{"phone_number_id":"123456789012"}}}]}]}`

	ctrl := gomock.NewController(t)
	tenantMock := mocks.NewMockTenantServiceInterface(ctrl)
	tenantMock.EXPECT().
		GetByWhatsAppPhoneID(gomock.Any(), "123456789012").
		Return(&dto.TenantDTO{ID: 2}, nil)

	cfg := config.Config{WhatsAppAppSecret: "segredo-meta"}

	app := novoAppComLogger()
	app.Post("/webhook", func(c *fiber.Ctx) error {
		ok, err := VerifyWebhookHandlerFiber(c, cfg, tenantMock)
		if !ok {
			return c.Status(401).SendString(err.Error())
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256="+assinaturaHMAC(t, []byte(body), "segredo-meta"))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestVerifyWebhookHandlerFiber_HMACInvalido(t *testing.T) {
	body := `{"entry":[{"changes":[{"value":{"metadata":{"phone_number_id":"123456789012"}}}]}]}`

	ctrl := gomock.NewController(t)
	tenantMock := mocks.NewMockTenantServiceInterface(ctrl)
	tenantMock.EXPECT().
		GetByWhatsAppPhoneID(gomock.Any(), "123456789012").
		Return(&dto.TenantDTO{ID: 2}, nil)

	cfg := config.Config{WhatsAppAppSecret: "segredo-meta"}

	app := novoAppComLogger()
	app.Post("/webhook", func(c *fiber.Ctx) error {
		ok, err := VerifyWebhookHandlerFiber(c, cfg, tenantMock)
		if !ok {
			return c.Status(401).SendString(err.Error())
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256=assinatura-errada")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestVerifyWebhookHandlerFiber_TenantNaoEncontrado(t *testing.T) {
	body := `{"entry":[{"changes":[{"value":{"metadata":{"phone_number_id":"999999999"}}}]}]}`

	ctrl := gomock.NewController(t)
	tenantMock := mocks.NewMockTenantServiceInterface(ctrl)
	tenantMock.EXPECT().
		GetByWhatsAppPhoneID(gomock.Any(), "999999999").
		Return(nil, assert.AnError)

	cfg := config.Config{WhatsAppAppSecret: "segredo-meta"}

	app := novoAppComLogger()
	app.Post("/webhook", func(c *fiber.Ctx) error {
		ok, err := VerifyWebhookHandlerFiber(c, cfg, tenantMock)
		if !ok {
			return c.Status(401).SendString(err.Error())
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256="+assinaturaHMAC(t, []byte(body), "segredo-meta"))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}
