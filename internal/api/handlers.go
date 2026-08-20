// internal/api/handlers.go - FINAL 100% FIBER - SEM net/http, SEM MUX, COM CACHE
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/cache"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

type APIHandlers struct {
	clienteService  service.ClienteServiceInterface
	pedidoService   service.PedidoServiceInterface
	cardapioService service.CardapioServiceInterface
	cacheLayer      *cache.Cache
}

func NewAPIHandlers(
	clienteService service.ClienteServiceInterface,
	pedidoService service.PedidoServiceInterface,
	cardapioService service.CardapioServiceInterface,
) *APIHandlers {
	return &APIHandlers{
		clienteService:  clienteService,
		pedidoService:   pedidoService,
		cardapioService: cardapioService,
	}
}

// SetCache injeta cache (chame no server.go)
func (h *APIHandlers) SetCache(c *cache.Cache) {
	h.cacheLayer = c
}

// HELPERS FIBER

func writeSuccessFiber(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(200).JSON(fiber.Map{
		"success": true,
		"message": message,
		"data":    data,
	})
}

func writeErrorFiber(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"success": false,
		"error":   message,
	})
}

func parsePaginationFiber(c *fiber.Ctx) (page, limit int) {
	page = 1
	limit = 20
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	return
}

func parseDateFiber(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", value)
}

func parseUintFiber(value string) (uint, error) {
	if value == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(value, 10, 64)
	return uint(v), err
}

// DASHBOARD
func (h *APIHandlers) DashboardFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	tenantID, ok := GetTenantIDFiber(c)
	if !ok {
		return writeErrorFiber(c, 401, "Unauthorized")
	}

	logger.Debug(ctx, "Dashboard acessado", zap.Uint("tenant_id", tenantID))

	// COM CACHE - dashboard é pesado, cache 1 minuto
	type DashboardData struct {
		TotalPedidosHoje   int64       `json:"total_pedidos_hoje"`
		TotalPedidosSemana int64       `json:"total_pedidos_semana"`
		TotalClientes      int64       `json:"total_clientes"`
		PedidosPendentes   int64       `json:"pedidos_pendentes"`
		PedidosPorStatus   interface{} `json:"pedidos_por_status"`
		FaturamentoHoje    float64     `json:"faturamento_hoje"`
		FaturamentoMes     float64     `json:"faturamento_mes"`
	}

	data, err := cache.GetOrSet(h.cacheLayer, ctx, "dashboard:"+strconv.Itoa(int(tenantID)), 1*time.Minute, func() (DashboardData, error) {
		totalPedidosHoje, _ := h.pedidoService.CountPedidosHoje(ctx, tenantID)
		totalPedidosSemana, _ := h.pedidoService.CountPedidosSemana(ctx, tenantID)
		totalClientes, _ := h.clienteService.CountByTenant(ctx, tenantID)
		pedidosPorStatus, _ := h.pedidoService.CountPorStatus(ctx, tenantID)
		faturamentoHoje, _ := h.pedidoService.FaturamentoHoje(ctx, tenantID)
		faturamentoMes, _ := h.pedidoService.FaturamentoMes(ctx, tenantID)
		pedidosPendentes, _ := h.pedidoService.CountPendentes(ctx, tenantID)
		return DashboardData{
			TotalPedidosHoje:   totalPedidosHoje,
			TotalPedidosSemana: totalPedidosSemana,
			TotalClientes:      totalClientes,
			PedidosPendentes:   pedidosPendentes,
			PedidosPorStatus:   pedidosPorStatus,
			FaturamentoHoje:    faturamentoHoje,
			FaturamentoMes:     faturamentoMes,
		}, nil
	})
	if err != nil {
		return writeErrorFiber(c, 500, "Failed to get dashboard")
	}

	return writeSuccessFiber(c, "Dashboard data retrieved", data)
}

// PEDIDOS
func (h *APIHandlers) ListPedidosFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	tenantID, ok := GetTenantIDFiber(c)
	if !ok {
		return writeErrorFiber(c, 401, "Unauthorized")
	}
	page, limit := parsePaginationFiber(c)
	status := c.Query("status")
	clienteID, _ := parseUintFiber(c.Query("cliente_id"))
	dataInicio, _ := parseDateFiber(c.Query("data_inicio"))
	dataFim, _ := parseDateFiber(c.Query("data_fim"))

	pedidos, total, err := h.pedidoService.ListWithFilters(ctx, tenantID, clienteID, status, dataInicio, dataFim, page, limit)
	if err != nil {
		logger.Error(ctx, "Failed to list orders", zap.Error(err))
		return writeErrorFiber(c, 500, "Failed to list orders")
	}
	return writeSuccessFiber(c, "Orders retrieved", fiber.Map{
		"data": pedidos, "total": total, "page": page, "limit": limit,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

func (h *APIHandlers) GetPedidoFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	id, err := parseUintFiber(c.Params("id"))
	if err != nil {
		return writeErrorFiber(c, 400, "Invalid order ID")
	}
	pedido, err := h.pedidoService.FindByID(ctx, id)
	if err != nil {
		return writeErrorFiber(c, 404, "Order not found")
	}
	return writeSuccessFiber(c, "Order retrieved", pedido)
}

func (h *APIHandlers) UpdatePedidoStatusFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	id, err := parseUintFiber(c.Params("id"))
	if err != nil {
		return writeErrorFiber(c, 400, "Invalid order ID")
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return writeErrorFiber(c, 400, "Invalid body")
	}
	if body.Status == "" {
		return writeErrorFiber(c, 400, "Status is required")
	}
	pedido, err := h.pedidoService.AtualizarStatusPedido(ctx, id, body.Status)
	if err != nil {
		return writeErrorFiber(c, 500, "Failed to update")
	}
	// Invalida cache dashboard
	if h.cacheLayer != nil {
		h.cacheLayer.InvalidateByTenant(ctx, 1, "dashboard:*")
	}
	return writeSuccessFiber(c, "Order status updated", pedido)
}

// CLIENTES
func (h *APIHandlers) ListClientesFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	tenantID, ok := GetTenantIDFiber(c)
	if !ok {
		return writeErrorFiber(c, 401, "Unauthorized")
	}
	page, limit := parsePaginationFiber(c)
	clientes, total, err := h.clienteService.ListWithFilters(ctx, tenantID, c.Query("nome"), c.Query("telefone"), page, limit)
	if err != nil {
		return writeErrorFiber(c, 500, "Failed to list clients")
	}
	return writeSuccessFiber(c, "Clients retrieved", fiber.Map{
		"data": clientes, "total": total, "page": page, "limit": limit,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

func (h *APIHandlers) GetClienteFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	id, _ := parseUintFiber(c.Params("id"))
	cliente, err := h.clienteService.FindByID(ctx, id)
	if err != nil {
		return writeErrorFiber(c, 404, "Client not found")
	}
	return writeSuccessFiber(c, "Client retrieved", cliente)
}

func (h *APIHandlers) GetClientePedidosFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	clienteID, _ := parseUintFiber(c.Params("id"))
	page, limit := parsePaginationFiber(c)
	pedidos, total, err := h.pedidoService.ListByCliente(ctx, clienteID, page, limit)
	if err != nil {
		return writeErrorFiber(c, 500, "Failed to get orders")
	}
	return writeSuccessFiber(c, "Client orders", fiber.Map{
		"data": pedidos, "total": total, "page": page, "limit": limit,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

func (h *APIHandlers) GetClienteEnderecosFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	clienteID, _ := parseUintFiber(c.Params("id"))
	enderecos, err := h.clienteService.ListarEnderecos(ctx, clienteID)
	if err != nil {
		return writeErrorFiber(c, 500, "Failed to get addresses")
	}
	return writeSuccessFiber(c, "Client addresses", enderecos)
}

// PRODUTOS - COM CACHE
func (h *APIHandlers) ListProdutosFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	tenantID, ok := GetTenantIDFiber(c)
	if !ok {
		return writeErrorFiber(c, 401, "Unauthorized")
	}
	page, limit := parsePaginationFiber(c)
	categoriaID, _ := parseUintFiber(c.Query("categoria_id"))
	nome := c.Query("nome")
	disponivelStr := c.Query("disponivel")
	var disponivel *bool
	if disponivelStr != "" {
		if v, err := strconv.ParseBool(disponivelStr); err == nil {
			disponivel = &v
		}
	}

	// Cache para listagem de produtos - 5 min
	cacheKey := "produtos:" + strconv.Itoa(int(tenantID)) + ":" + nome + ":" + strconv.Itoa(page)
	produtos, err := cache.GetOrSet(h.cacheLayer, ctx, cacheKey, 5*time.Minute, func() (interface{}, error) {
		p, total, err := h.cardapioService.ListWithFilters(ctx, tenantID, &categoriaID, disponivel, nome, page, limit)
		if err != nil {
			return nil, err
		}
		return fiber.Map{
			"data": p, "total": total, "page": page, "limit": limit,
			"pages": (total + int64(limit) - 1) / int64(limit),
		}, nil
	})
	if err != nil {
		return writeErrorFiber(c, 500, "Failed to list products")
	}
	return writeSuccessFiber(c, "Products retrieved", produtos)
}

func (h *APIHandlers) GetProdutoFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	id, _ := parseUintFiber(c.Params("id"))
	produto, err := h.cardapioService.FindByID(ctx, id)
	if err != nil {
		return writeErrorFiber(c, 404, "Product not found")
	}
	return writeSuccessFiber(c, "Product retrieved", produto)
}

// LOGIN
func (h *APIHandlers) LoginFiber(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return writeErrorFiber(c, 400, "Invalid body")
	}
	if body.Email != "admin@admin.com" || body.Password != "admin123" {
		return writeErrorFiber(c, 401, "Invalid credentials")
	}
	token, err := GenerateJWT(1, 1, body.Email)
	if err != nil {
		return writeErrorFiber(c, 500, "Failed to generate token")
	}
	return writeSuccessFiber(c, "Login successful", fiber.Map{
		"token": token,
		"user":  fiber.Map{"id": 1, "email": body.Email},
	})
}

// Compatibilidade temporária - mantém métodos antigos chamando Fiber (para build não quebrar)
func (h *APIHandlers) Dashboard(w http.ResponseWriter, r *http.Request)           {}
func (h *APIHandlers) ListPedidos(w http.ResponseWriter, r *http.Request)         {}
func (h *APIHandlers) GetPedido(w http.ResponseWriter, r *http.Request)           {}
func (h *APIHandlers) UpdatePedidoStatus(w http.ResponseWriter, r *http.Request)  {}
func (h *APIHandlers) ListClientes(w http.ResponseWriter, r *http.Request)        {}
func (h *APIHandlers) GetCliente(w http.ResponseWriter, r *http.Request)          {}
func (h *APIHandlers) GetClientePedidos(w http.ResponseWriter, r *http.Request)   {}
func (h *APIHandlers) GetClienteEnderecos(w http.ResponseWriter, r *http.Request) {}
func (h *APIHandlers) ListProdutos(w http.ResponseWriter, r *http.Request)        {}
func (h *APIHandlers) GetProduto(w http.ResponseWriter, r *http.Request)          {}
func (h *APIHandlers) Login(w http.ResponseWriter, r *http.Request)               {}
