package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/server/response"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"github.com/rafapasa/mcp-server-openerp/internal/webhook"
	"go.uber.org/zap"
)

type APIHandlers struct {
	authService           service.AuthServiceInterface
	clienteService        service.ClienteServiceInterface
	pedidoService         service.PedidoServiceInterface
	cardapioService       service.CardapioServiceInterface
	formaPagamentoService service.FormaPagamentoServiceInterface
	tenantService         service.TenantServiceInterface
	cache                 database.RedisInterface
	whatsAppClient        *webhook.WhatsAppClient
}

func NewAPIHandlers(
	authService service.AuthServiceInterface,
	clienteService service.ClienteServiceInterface,
	pedidoService service.PedidoServiceInterface,
	cardapioService service.CardapioServiceInterface,
	formaPagamentoService service.FormaPagamentoServiceInterface,
	tenantService service.TenantServiceInterface,
	cache database.RedisInterface,
	whatsAppClient *webhook.WhatsAppClient,
) *APIHandlers {
	return &APIHandlers{
		authService:           authService,
		clienteService:        clienteService,
		pedidoService:         pedidoService,
		cardapioService:       cardapioService,
		formaPagamentoService: formaPagamentoService,
		tenantService:         tenantService,
		cache:                 cache,
		whatsAppClient:        whatsAppClient,
	}
}

// POST /api/v1/login — formato LoginResponseList sem envelope {data:}
func (h *APIHandlers) LoginFiber(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ValidationError(c, "invalid body")
	}
	loginResponse, err := h.authService.Authenticate(c.Context(), req)
	if err != nil {
		return response.FromError(c, err)
	}
	logger.Info(c.Context(), "Login efetuado", zap.Any("loginResponse", loginResponse))
	return c.Status(fiber.StatusOK).JSON(loginResponse)
}

// GET /api/v1/dashboard
func (h *APIHandlers) DashboardFiber(c *fiber.Ctx) error {
	tenantID, ok := response.GetTenantIDOrRespond(c)
	if !ok {
		return nil
	}
	stats, err := h.pedidoService.FindByTenant(c.Context(), tenantID)
	if err != nil {
		logger.GetLogger().Error("dashboard error", zap.Error(err))
		return response.FromError(c, err)
	}
	return response.OK(c, stats)
}

// GET /api/v1/pedidos?page=1&limit=20
func (h *APIHandlers) ListPedidosFiber(c *fiber.Ctx) error {
	tenantID, ok := response.GetTenantIDOrRespond(c)
	if !ok {
		return nil
	}
	pag := response.GetPaginatedRequest(c)
	pedidos, total, err := h.pedidoService.FindByTenantPaginated(c.Context(), tenantID, pag.Page, pag.Limit)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.Paginated(c, pedidos, total, pag.Page, pag.Limit)
}

// GET /api/v1/pedidos/:id
func (h *APIHandlers) GetPedidoFiber(c *fiber.Ctx) error {
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	pedido, err := h.pedidoService.FindByID(c.Context(), id)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, pedido)
}

// PATCH /api/v1/pedidos/:id/status
func (h *APIHandlers) UpdatePedidoStatusFiber(c *fiber.Ctx) error {
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.ValidationError(c, "invalid body")
	}
	pedido, err := h.pedidoService.AtualizarStatusPedido(c.Context(), id, req.Status)
	if err != nil {
		return response.FromError(c, err)
	}
	if models.NormalizarStatusPedido(req.Status) == models.StatusSaiuParaEntrega {
		if err := h.notificarPedidoSaiuParaEntrega(c.Context(), pedido); err != nil {
			logger.Warn(c.Context(), "Erro ao enviar notificação de pedido saiu para entrega", zap.Error(err), zap.Uint("pedido_id", pedido.ID))
		}
	}
	return response.OK(c, pedido)
}

func (h *APIHandlers) notificarPedidoSaiuParaEntrega(ctx context.Context, pedido *dto.PedidoDTO) error {
	if pedido == nil || h.whatsAppClient == nil || pedido.ClienteTelefone == "" {
		return nil
	}
	if h.cache != nil {
		key := fmt.Sprintf("notificacao:saiu:%d", pedido.ID)
		set, err := h.cache.SetNXWithContext(ctx, key, "1", 24*time.Hour)
		if err != nil {
			return err
		}
		if !set {
			logger.Info(ctx, "Notificação de entrega já enviada; ignorando duplicata", zap.Uint("pedido_id", pedido.ID))
			return nil
		}
	}
	msg := formatarMensagemSaiuParaEntregaDTO(pedido)
	if err := h.whatsAppClient.SendMessageCtx(ctx, pedido.ClienteTelefone, msg); err != nil {
		return err
	}
	logger.Info(ctx, "Notificação de pedido saiu para entrega enviada com sucesso", zap.Uint("pedido_id", pedido.ID), zap.String("telefone", pedido.ClienteTelefone))
	return nil
}

func formatarMensagemSaiuParaEntregaDTO(pedido *dto.PedidoDTO) string {
	if pedido == nil {
		return ""
	}
	clienteNome := strings.TrimSpace(pedido.ClienteNome)
	if clienteNome == "" {
		clienteNome = "Cliente"
	}
	msg := fmt.Sprintf("🛵 %s, seu pedido #%d saiu para entrega! Chega em ~15 min.", clienteNome, pedido.ID)
	if pedido.EnderecoEntrega != nil {
		endereco := strings.TrimSpace(pedido.EnderecoEntrega.Logradouro)
		if pedido.EnderecoEntrega.Numero != "" {
			if endereco != "" {
				endereco += ", " + pedido.EnderecoEntrega.Numero
			} else {
				endereco = pedido.EnderecoEntrega.Numero
			}
		}
		if pedido.EnderecoEntrega.Bairro != "" {
			if endereco != "" {
				endereco += " - " + pedido.EnderecoEntrega.Bairro
			} else {
				endereco = pedido.EnderecoEntrega.Bairro
			}
		}
		if endereco != "" {
			msg += fmt.Sprintf(" Endereço: %s", endereco)
		}
	}
	return msg
}

// GET /api/v1/clientes
func (h *APIHandlers) ListClientesFiber(c *fiber.Ctx) error {
	tenantID, ok := response.GetTenantIDOrRespond(c)
	if !ok {
		return nil
	}
	pag := response.GetPaginatedRequest(c)
	clientes, total, err := h.clienteService.FindByTenantPaginated(c.Context(), tenantID, pag.Page, pag.Limit)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.Paginated(c, clientes, total, pag.Page, pag.Limit)
}

func (h *APIHandlers) GetClienteFiber(c *fiber.Ctx) error {
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	cliente, err := h.clienteService.FindByID(c.Context(), id)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, cliente)
}

func (h *APIHandlers) GetClientePedidosFiber(c *fiber.Ctx) error {
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	pag := response.GetPaginatedRequest(c)
	pedidos, total, err := h.pedidoService.ListByCliente(c.Context(), id, pag.Page, pag.Limit)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.Paginated(c, pedidos, total, pag.Page, pag.Limit)
}

func (h *APIHandlers) GetClienteEnderecosFiber(c *fiber.Ctx) error {
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	enderecos, err := h.clienteService.ListarEnderecos(c.Context(), id)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, enderecos)
}

// GET /api/v1/produtos
func (h *APIHandlers) ListProdutosFiber(c *fiber.Ctx) error {
	tenantID, ok := response.GetTenantIDOrRespond(c)
	if !ok {
		return nil
	}
	pag := response.GetPaginatedRequest(c)
	produtos, total, err := h.cardapioService.FindByTenantPaginated(c.Context(), tenantID, pag.Page, pag.Limit)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.Paginated(c, produtos, total, pag.Page, pag.Limit)
}

func (h *APIHandlers) GetProdutoFiber(c *fiber.Ctx) error {
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	produto, err := h.cardapioService.FindByID(c.Context(), id)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, produto)
}

// Formas pagamento
func (h *APIHandlers) ListFormasPagamentoFiber(c *fiber.Ctx) error {
	tenantID, ok := response.GetTenantIDOrRespond(c)
	if !ok {
		return nil
	}
	apenasAtivas := c.Query("ativas", "true") != "false"
	formas, err := h.formaPagamentoService.Listar(c.Context(), tenantID, apenasAtivas)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, formas)
}

func (h *APIHandlers) GetFormaPagamentoFiber(c *fiber.Ctx) error {
	tenantID, ok := response.GetTenantIDOrRespond(c)
	if !ok {
		return nil
	}
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	forma, err := h.formaPagamentoService.Buscar(c.Context(), tenantID, id)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, forma)
}

func (h *APIHandlers) CreateFormaPagamentoFiber(c *fiber.Ctx) error {
	tenantID, ok := response.GetTenantIDOrRespond(c)
	if !ok {
		return nil
	}
	var req dto.CriarFormaPagamentoRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ValidationError(c, "invalid body")
	}
	forma, err := h.formaPagamentoService.Criar(c.Context(), tenantID, req)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.Created(c, forma)
}

func (h *APIHandlers) UpdateFormaPagamentoFiber(c *fiber.Ctx) error {
	tenantID, ok := response.GetTenantIDOrRespond(c)
	if !ok {
		return nil
	}
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	var req dto.AtualizarFormaPagamentoRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ValidationError(c, "invalid body")
	}
	forma, err := h.formaPagamentoService.Atualizar(c.Context(), tenantID, id, req)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, forma)
}

func (h *APIHandlers) DeleteFormaPagamentoFiber(c *fiber.Ctx) error {
	tenantID, ok := response.GetTenantIDOrRespond(c)
	if !ok {
		return nil
	}
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	if err := h.formaPagamentoService.Inativar(c.Context(), tenantID, id); err != nil {
		return response.FromError(c, err)
	}
	return response.Deactivated(c)
}

// TENANTS
func (h *APIHandlers) ListTenantsFiber(c *fiber.Ctx) error {
	tenants, err := h.tenantService.List(c.Context())
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, tenants)
}

func (h *APIHandlers) GetTenantFiber(c *fiber.Ctx) error {
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	tenant, err := h.tenantService.GetByID(c.Context(), id)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, tenant)
}

func (h *APIHandlers) CreateTenantFiber(c *fiber.Ctx) error {
	var req dto.CreateTenantDTO
	if err := c.BodyParser(&req); err != nil {
		return response.ValidationError(c, "invalid body: "+err.Error())
	}
	if req.Nome == "" {
		return response.ValidationError(c, "nome é obrigatório")
	}
	created, err := h.tenantService.Create(c.Context(), req)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.Created(c, created)
}

func (h *APIHandlers) UpdateTenantFiber(c *fiber.Ctx) error {
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	var req dto.UpdateTenantDTO
	if err := c.BodyParser(&req); err != nil {
		return response.ValidationError(c, "invalid body")
	}
	updated, err := h.tenantService.Update(c.Context(), id, req)
	if err != nil {
		return response.FromError(c, err)
	}
	return response.OK(c, updated)
}

func (h *APIHandlers) DeleteTenantFiber(c *fiber.Ctx) error {
	id, ok := response.ParseIDParamOrRespond(c, "id")
	if !ok {
		return nil
	}
	if err := h.tenantService.Delete(c.Context(), id); err != nil {
		return response.FromError(c, err)
	}
	return response.Deleted(c, id)
}
