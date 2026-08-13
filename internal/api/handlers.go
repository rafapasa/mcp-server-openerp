// internal/api/handlers.go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// APIHandlers contém todos os handlers
type APIHandlers struct {
	clienteService  service.ClienteServiceInterface
	pedidoService   service.PedidoServiceInterface
	cardapioService service.CardapioServiceInterface
}

// NewAPIHandlers cria uma nova instância
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

// ============================================
// HELPERS
// ============================================

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error(context.Background(), "Failed to write JSON response", zap.Error(err))
	}
}

func writeSuccess(w http.ResponseWriter, message string, data interface{}) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": message,
		"data":    data,
	})
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

func parsePagination(r *http.Request) (page, limit int) {
	page = 1
	limit = 20

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	return page, limit
}

func parseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", value)
}

func parseUint(value string) (uint, error) {
	if value == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(value, 10, 64)
	return uint(v), err
}

// ============================================
// 1. DASHBOARD
// ============================================

// Dashboard GET /api/v1/dashboard
func (h *APIHandlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, ok := GetTenantID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	logger.Debug(ctx, "Dashboard acessado", zap.Uint("tenant_id", tenantID))

	totalPedidosHoje, _ := h.pedidoService.CountPedidosHoje(ctx, tenantID)
	totalPedidosSemana, _ := h.pedidoService.CountPedidosSemana(ctx, tenantID)
	totalClientes, _ := h.clienteService.CountByTenant(ctx, tenantID)
	pedidosPorStatus, _ := h.pedidoService.CountPorStatus(ctx, tenantID)
	faturamentoHoje, _ := h.pedidoService.FaturamentoHoje(ctx, tenantID)
	faturamentoMes, _ := h.pedidoService.FaturamentoMes(ctx, tenantID)
	pedidosPendentes, _ := h.pedidoService.CountPendentes(ctx, tenantID)

	writeSuccess(w, "Dashboard data retrieved", map[string]interface{}{
		"total_pedidos_hoje":   totalPedidosHoje,
		"total_pedidos_semana": totalPedidosSemana,
		"total_clientes":       totalClientes,
		"pedidos_pendentes":    pedidosPendentes,
		"pedidos_por_status":   pedidosPorStatus,
		"faturamento_hoje":     faturamentoHoje,
		"faturamento_mes":      faturamentoMes,
	})
}

// ============================================
// 2. PEDIDOS
// ============================================

// ListPedidos GET /api/v1/pedidos
func (h *APIHandlers) ListPedidos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, ok := GetTenantID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	page, limit := parsePagination(r)
	status := r.URL.Query().Get("status")
	clienteID, _ := parseUint(r.URL.Query().Get("cliente_id"))
	dataInicio, _ := parseDate(r.URL.Query().Get("data_inicio"))
	dataFim, _ := parseDate(r.URL.Query().Get("data_fim"))

	logger.Debug(ctx, "Listando pedidos",
		zap.Uint("tenant_id", tenantID),
		zap.String("status", status),
		zap.Uint("cliente_id", clienteID),
	)

	pedidos, total, err := h.pedidoService.ListWithFilters(ctx, tenantID, clienteID, status, dataInicio, dataFim, page, limit)
	if err != nil {
		logger.Error(ctx, "Failed to list orders", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to list orders")
		return
	}

	writeSuccess(w, "Orders retrieved successfully", map[string]interface{}{
		"data":  pedidos,
		"total": total,
		"page":  page,
		"limit": limit,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetPedido GET /api/v1/pedidos/{id}
func (h *APIHandlers) GetPedido(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseUint(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	logger.Debug(ctx, "Buscando pedido", zap.Uint("id", id))

	pedido, err := h.pedidoService.FindByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "Failed to get order", zap.Uint("id", id), zap.Error(err))
		writeError(w, http.StatusNotFound, "Order not found")
		return
	}

	writeSuccess(w, "Order retrieved successfully", pedido)
}

// UpdatePedidoStatus PATCH /api/v1/pedidos/{id}/status
func (h *APIHandlers) UpdatePedidoStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseUint(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if body.Status == "" {
		writeError(w, http.StatusBadRequest, "Status is required")
		return
	}

	logger.Info(ctx, "Atualizando status do pedido",
		zap.Uint("id", id),
		zap.String("status", body.Status),
	)

	pedido, err := h.pedidoService.AtualizarStatusPedido(ctx, id, body.Status)
	if err != nil {
		logger.Error(ctx, "Failed to update order status", zap.Uint("id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to update order status")
		return
	}

	writeSuccess(w, "Order status updated successfully", pedido)
}

// ============================================
// 3. CLIENTES
// ============================================

// ListClientes GET /api/v1/clientes
func (h *APIHandlers) ListClientes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, ok := GetTenantID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	page, limit := parsePagination(r)
	nome := r.URL.Query().Get("nome")
	telefone := r.URL.Query().Get("telefone")

	logger.Debug(ctx, "Listando clientes",
		zap.Uint("tenant_id", tenantID),
		zap.String("nome", nome),
		zap.String("telefone", telefone),
	)

	clientes, total, err := h.clienteService.ListWithFilters(ctx, tenantID, nome, telefone, page, limit)
	if err != nil {
		logger.Error(ctx, "Failed to list clients", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to list clients")
		return
	}

	writeSuccess(w, "Clients retrieved successfully", map[string]interface{}{
		"data":  clientes,
		"total": total,
		"page":  page,
		"limit": limit,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetCliente GET /api/v1/clientes/{id}
func (h *APIHandlers) GetCliente(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseUint(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid client ID")
		return
	}

	logger.Debug(ctx, "Buscando cliente", zap.Uint("id", id))

	cliente, err := h.clienteService.FindByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "Failed to get client", zap.Uint("id", id), zap.Error(err))
		writeError(w, http.StatusNotFound, "Client not found")
		return
	}

	writeSuccess(w, "Client retrieved successfully", cliente)
}

// GetClientePedidos GET /api/v1/clientes/{id}/pedidos
func (h *APIHandlers) GetClientePedidos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clienteID, err := parseUint(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid client ID")
		return
	}

	page, limit := parsePagination(r)

	logger.Debug(ctx, "Buscando pedidos do cliente",
		zap.Uint("cliente_id", clienteID),
	)

	pedidos, total, err := h.pedidoService.ListByCliente(ctx, clienteID, page, limit)
	if err != nil {
		logger.Error(ctx, "Failed to get client orders", zap.Uint("cliente_id", clienteID), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to get client orders")
		return
	}

	writeSuccess(w, "Client orders retrieved successfully", map[string]interface{}{
		"data":  pedidos,
		"total": total,
		"page":  page,
		"limit": limit,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetClienteEnderecos GET /api/v1/clientes/{id}/enderecos
func (h *APIHandlers) GetClienteEnderecos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clienteID, err := parseUint(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid client ID")
		return
	}

	logger.Debug(ctx, "Buscando endereços do cliente",
		zap.Uint("cliente_id", clienteID),
	)

	enderecos, err := h.clienteService.ListarEnderecos(ctx, clienteID)
	if err != nil {
		logger.Error(ctx, "Failed to get client addresses", zap.Uint("cliente_id", clienteID), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to get client addresses")
		return
	}

	writeSuccess(w, "Client addresses retrieved successfully", enderecos)
}

// ============================================
// 4. PRODUTOS
// ============================================

// ListProdutos GET /api/v1/produtos
func (h *APIHandlers) ListProdutos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, ok := GetTenantID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	page, limit := parsePagination(r)
	categoriaID, _ := parseUint(r.URL.Query().Get("categoria_id"))
	disponivelStr := r.URL.Query().Get("disponivel")
	nome := r.URL.Query().Get("nome")

	var disponivel *bool
	if disponivelStr != "" {
		if v, err := strconv.ParseBool(disponivelStr); err == nil {
			disponivel = &v
		}
	}

	logger.Debug(ctx, "Listando produtos",
		zap.Uint("tenant_id", tenantID),
		zap.Uint("categoria_id", categoriaID),
		zap.String("nome", nome),
	)

	produtos, total, err := h.cardapioService.ListWithFilters(ctx, tenantID, &categoriaID, disponivel, nome, page, limit)
	if err != nil {
		logger.Error(ctx, "Failed to list products", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to list products")
		return
	}

	writeSuccess(w, "Products retrieved successfully", map[string]interface{}{
		"data":  produtos,
		"total": total,
		"page":  page,
		"limit": limit,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetProduto GET /api/v1/produtos/{id}
func (h *APIHandlers) GetProduto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseUint(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	logger.Debug(ctx, "Buscando produto", zap.Uint("id", id))

	produto, err := h.cardapioService.FindByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "Failed to get product", zap.Uint("id", id), zap.Error(err))
		writeError(w, http.StatusNotFound, "Product not found")
		return
	}

	writeSuccess(w, "Product retrieved successfully", produto)
}

// ============================================
// 0. LOGIN
// ============================================

// Login POST /api/v1/login
func (h *APIHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logger.Warn(r.Context(), "Erro ao decodificar login", zap.Error(err))
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	logger.Debug(r.Context(), "Tentativa de login", zap.String("email", body.Email))

	// TODO: Implementar validação real (buscar usuário no banco)
	if body.Email != "admin@admin.com" || body.Password != "admin123" {
		logger.Warn(r.Context(), "Falha no login", zap.String("email", body.Email))
		writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, err := GenerateJWT(1, 1, body.Email)
	if err != nil {
		logger.Error(r.Context(), "Failed to generate JWT", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	logger.Info(r.Context(), "Login bem-sucedido", zap.String("email", body.Email))

	writeSuccess(w, "Login successful", map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":    1,
			"email": body.Email,
		},
	})
}
