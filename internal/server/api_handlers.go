package server

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

type APIHandlers struct {
	authService           service.AuthServiceInterface
	clienteService        service.ClienteServiceInterface
	pedidoService         service.PedidoServiceInterface
	cardapioService       service.CardapioServiceInterface
	formaPagamentoService service.FormaPagamentoServiceInterface
}

func NewAPIHandlers(
	authService service.AuthServiceInterface,
	clienteService service.ClienteServiceInterface,
	pedidoService service.PedidoServiceInterface,
	cardapioService service.CardapioServiceInterface,
	formaPagamentoService service.FormaPagamentoServiceInterface,
) *APIHandlers {
	return &APIHandlers{
		authService: authService, clienteService: clienteService,
		pedidoService: pedidoService, cardapioService: cardapioService,
		formaPagamentoService: formaPagamentoService,
	}
}

// POST /api/v1/login
func (h *APIHandlers) LoginFiber(c *fiber.Ctx) error {
	var loginRequest dto.LoginRequest
	if err := c.BodyParser(&loginRequest); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	loginResponse, err := h.authService.Authenticate(c.Context(), loginRequest)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	logger.Info(c.Context(), "Login efetuado com sucesso", zap.Any("loginResponse", loginResponse))
	return c.JSON(loginResponse)
}

// GET /api/v1/dashboard
func (h *APIHandlers) DashboardFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	stats, err := h.pedidoService.FindByTenant(c.Context(), tenantID)
	if err != nil {
		logger.GetLogger().Error("dashboard error", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "internal"})
	}
	return c.JSON(stats)
}

// GET /api/v1/pedidos
func (h *APIHandlers) ListPedidosFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	pedidos, err := h.pedidoService.FindByTenant(c.Context(), tenantID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(pedidos)
}

// GET /api/v1/pedidos/:id
func (h *APIHandlers) GetPedidoFiber(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	pedido, err := h.pedidoService.FindByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(pedido)
}

// PATCH /api/v1/pedidos/:id/status
func (h *APIHandlers) UpdatePedidoStatusFiber(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if _, err := h.pedidoService.AtualizarStatusPedido(c.Context(), uint(id), req.Status); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "updated"})
}

// GET /api/v1/clientes
func (h *APIHandlers) ListClientesFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	clientes, err := h.clienteService.FindByTenant(c.Context(), tenantID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(clientes)
}

// GET /api/v1/clientes/:id
func (h *APIHandlers) GetClienteFiber(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	cliente, err := h.clienteService.FindByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(cliente)
}

// GET /api/v1/clientes/:id/pedidos
func (h *APIHandlers) GetClientePedidosFiber(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	pedidos, total, err := h.pedidoService.ListByCliente(c.Context(), uint(id), page, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	resp := dto.PedidoListResponseDTO{
		Pedidos: pedidos, Total: total, Page: page, Limit: limit,
		TotalPages: (total + int64(limit) - 1) / int64(limit),
	}
	return c.JSON(resp)
}

// GET /api/v1/clientes/:id/enderecos
func (h *APIHandlers) GetClienteEnderecosFiber(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	enderecos, err := h.clienteService.ListarEnderecos(c.Context(), uint(id))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(enderecos)
}

// GET /api/v1/produtos
func (h *APIHandlers) ListProdutosFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	produtos, err := h.cardapioService.GetCardapio(c.Context(), tenantID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(produtos)
}

// GET /api/v1/produtos/:id
func (h *APIHandlers) GetProdutoFiber(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	produto, err := h.cardapioService.FindByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(produto)
}

// === NOVO dev-11: Produto CRUD com invalidação ===

// POST /api/v1/produtos
func (h *APIHandlers) CreateProdutoFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	var req models.Produto
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	req.TenantID = tenantID
	if err := h.cardapioService.Create(c.Context(), &req); err != nil {
		logger.Error(c.Context(), "erro criar produto", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(req)
}

// PUT /api/v1/produtos/:id - FIX dev-11 invalida cache
func (h *APIHandlers) UpdateProdutoFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	id, _ := strconv.Atoi(c.Params("id"))
	var req models.Produto
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	req.ID = uint(id)
	req.TenantID = tenantID

	// busca existente para garantir ownership
	existing, err := h.cardapioService.FindByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	if existing.TenantID != tenantID {
		return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
	}

	// merge simples: mantém campos não enviados? Para simplificar, atualiza o model
	produtoModel := models.Produto{
		ID: req.ID, TenantID: tenantID, CategoriaID: req.CategoriaID,
		Nome: req.Nome, Descricao: req.Descricao, Preco: req.Preco,
		Ingredientes: req.Ingredientes, Disponivel: req.Disponivel,
		TempoPreparo: req.TempoPreparo,
	}
	if err := h.cardapioService.Update(c.Context(), &produtoModel); err != nil {
		logger.Error(c.Context(), "erro atualizar produto", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "updated", "id": id})
}

// DELETE /api/v1/produtos/:id
func (h *APIHandlers) DeleteProdutoFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	id, _ := strconv.Atoi(c.Params("id"))
	if err := h.cardapioService.Delete(c.Context(), uint(id), tenantID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "deleted"})
}

// PATCH /api/v1/produtos/:id/disponibilidade
func (h *APIHandlers) UpdateDisponibilidadeFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Disponivel bool `json:"disponivel"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.cardapioService.UpdateDisponibilidade(c.Context(), uint(id), tenantID, req.Disponivel); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "updated"})
}

func (h *APIHandlers) ListFormasPagamentoFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	apenasAtivas := c.Query("ativas", "true") != "false"
	formas, err := h.formaPagamentoService.Listar(c.Context(), tenantID, apenasAtivas)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(formas)
}

func (h *APIHandlers) GetFormaPagamentoFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "id inválido"})
	}
	forma, err := h.formaPagamentoService.Buscar(c.Context(), tenantID, uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(forma)
}

func (h *APIHandlers) CreateFormaPagamentoFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	var req dto.CriarFormaPagamentoRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	forma, err := h.formaPagamentoService.Criar(c.Context(), tenantID, req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(forma)
}

func (h *APIHandlers) UpdateFormaPagamentoFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "id inválido"})
	}
	var req dto.AtualizarFormaPagamentoRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	forma, err := h.formaPagamentoService.Atualizar(c.Context(), tenantID, uint(id), req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(forma)
}

func (h *APIHandlers) DeleteFormaPagamentoFiber(c *fiber.Ctx) error {
	tenantID, err := h.getTenantIdByHeaderFiber(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"erro": err.Error()})
	}
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "id inválido"})
	}
	if err := h.formaPagamentoService.Inativar(c.Context(), tenantID, uint(id)); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "deactivated"})
}

func (h *APIHandlers) getTenantIdByHeaderFiber(c *fiber.Ctx) (uint, error) {
	tenantID, err := strconv.Atoi(c.Get("X-Tenant-ID"))
	if err != nil {
		return 0, err
	}
	return uint(tenantID), nil
}
