// internal/webhook/handlers_test.go
package webhook

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/mocks"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// novoAppComLogger monta um app Fiber cujo contexto já carrega um logger nop
// (evita depender do logger global, que só é inicializado no entry point).
func novoAppComLogger() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.SetUserContext(logger.WithLogger(context.Background(), zap.NewNop()))
		return c.Next()
	})
	return app
}

func novoHandlerComMock(t *testing.T) (*WebhookHandler, *mocks.MockTenantServiceInterface) {
	t.Helper()
	ctrl := gomock.NewController(t)
	tenantMock := mocks.NewMockTenantServiceInterface(ctrl)
	h := NewWebhookHandler(&config.Config{}, &Processor{}, tenantMock)
	return h, tenantMock
}

func TestHandleWebhookFiber_GET_Subscribe_ValidaNoBanco(t *testing.T) {
	h, tenantMock := novoHandlerComMock(t)
	app := novoAppComLogger()
	app.Get("/webhook", h.HandleWebhookFiber)

	// verify_token agora vem da tabela tenants (whatsapp_verify_token)
	tenantMock.EXPECT().GetByVerifyToken(gomock.Any(), "token-seguro").Return(&dto.TenantDTO{ID: 2}, nil)

	req := httptest.NewRequest("GET", "/webhook?hub.mode=subscribe&hub.verify_token=token-seguro&hub.challenge=ch123", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "ch123", string(body))
}

func TestHandleWebhookFiber_GET_Subscribe_TokenInvalido(t *testing.T) {
	h, tenantMock := novoHandlerComMock(t)
	app := novoAppComLogger()
	app.Get("/webhook", h.HandleWebhookFiber)

	tenantMock.EXPECT().GetByVerifyToken(gomock.Any(), "token-ruim").Return(nil, assert.AnError)

	req := httptest.NewRequest("GET", "/webhook?hub.mode=subscribe&hub.verify_token=token-ruim&hub.challenge=ch123", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestHandleWebhookFiber_GET_Subscribe_ModeDiferente(t *testing.T) {
	h, _ := novoHandlerComMock(t)
	app := novoAppComLogger()
	app.Get("/webhook", h.HandleWebhookFiber)

	// modo != subscribe nem consulta o banco
	req := httptest.NewRequest("GET", "/webhook?hub.mode=outro&hub.verify_token=x&hub.challenge=ch", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestHandleWebhookFiber_POST_AssinaturaInvalidaRetornaForbidden(t *testing.T) {
	body := `{"entry":[{"changes":[{"value":{"metadata":{"phone_number_id":"123456789012"}}}]}]}`

	ctrl := gomock.NewController(t)
	tenantMock := mocks.NewMockTenantServiceInterface(ctrl)
	tenantMock.EXPECT().
		GetByWhatsAppPhoneID(gomock.Any(), "123456789012").
		Return(&dto.TenantDTO{ID: 2}, nil)

	h := NewWebhookHandler(
		&config.Config{WhatsAppAppSecret: "segredo-meta"},
		&Processor{},
		tenantMock,
	)
	app := novoAppComLogger()
	app.Post("/webhook", h.HandleWebhookFiber)

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=assinatura-invalida")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}
