package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/server/response"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

type APIHandlers struct {
	authService           service.AuthServiceInterface
	clienteService        service.ClienteServiceInterface
	pedidoService         service.PedidoServiceInterface
	cardapioService       service.CardapioServiceInterface
	formaPagamentoService service.FormaPagamentoServiceInterface
	tenantService         service.TenantServiceInterface
}

func NewAPIHandlers(
	authService service.AuthServiceInterface,
	clienteService service.ClienteServiceInterface,
	pedidoService service.PedidoServiceInterface,
	cardapioService service.CardapioServiceInterface,
	formaPagamentoService service.FormaPagamentoServiceInterface,
	tenantService service.TenantServiceInterface,
) *APIHandlers {
	return &APIHandlers{
		authService:           authService,
		clienteService:        clienteService,
		pedidoService:         pedidoService,
		cardapioService:       cardapioService,
		formaPagamentoService: formaPagamentoService,
		tenantService:         tenantService,
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
	return response.OK(c, pedido)
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
