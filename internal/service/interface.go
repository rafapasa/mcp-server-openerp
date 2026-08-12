// internal/service/interfaces.go
package service

import (
	"context"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

// CarrinhoServiceInterface define as operações do serviço de carrinho
type CarrinhoServiceInterface interface {
	AdicionarItem(ctx context.Context, clienteID, tenantID uint, item dto.ItemCarrinho) error
	RemoverItem(ctx context.Context, clienteID, tenantID uint, nome string, quantidade int) error
	GetCarrinho(ctx context.Context, clienteID, tenantID uint) (*dto.Carrinho, error)
	LimparCarrinho(ctx context.Context, clienteID, tenantID uint) error
	FinalizarCarrinho(ctx context.Context, clienteID, tenantID uint, clienteNome string) (*dto.PedidoConfirmado, error)
	CalcularTotal(carrinho *dto.Carrinho) float64
	CalcularTempoEstimado(carrinho *dto.Carrinho) int
}

// ============================================
// CARDAPIO SERVICE
// ============================================

type CardapioServiceInterface interface {
	// Existing methods
	GetCardapio(ctx context.Context, tenantID uint) ([]dto.ProdutoItem, error)
	BuscarProdutoPorNome(ctx context.Context, tenantID string, nome string) (*dto.ProdutoItem, error)
	ItemExisteNoCardapio(cardapio []dto.ProdutoItem, nome string) (bool, float64)
	EncontrarItemSimilar(cardapio []dto.ProdutoItem, nome string) string
	FormatarCardapio(cardapio []dto.ProdutoItem) string

	// NEW: Dashboard methods
	ListWithFilters(ctx context.Context, tenantID uint, categoriaID *uint, disponivel *bool, nome string, page, limit int) ([]dto.ProdutoDTO, int64, error)
	FindByID(ctx context.Context, id uint) (*dto.ProdutoDTO, error)
}

// ============================================
// PEDIDO SERVICE
// ============================================

type PedidoServiceInterface interface {
	// Existing methods
	ProcessarPedido(ctx context.Context, tenantID, clienteID uint, clienteNome string, pedidoExtraido *dto.PedidoExtraido) (*dto.PedidoConfirmado, error)
	FindByTenant(ctx context.Context, tenantID uint) ([]dto.PedidoDTO, error)

	// NEW: Dashboard methods
	CountPedidosHoje(ctx context.Context, tenantID uint) (int64, error)
	CountPedidosSemana(ctx context.Context, tenantID uint) (int64, error)
	CountPorStatus(ctx context.Context, tenantID uint) (map[string]int64, error)
	CountPendentes(ctx context.Context, tenantID uint) (int64, error)
	FaturamentoHoje(ctx context.Context, tenantID uint) (float64, error)
	FaturamentoMes(ctx context.Context, tenantID uint) (float64, error)

	// NEW: List methods
	ListWithFilters(ctx context.Context, tenantID uint, clienteID uint, status string, dataInicio, dataFim time.Time, page, limit int) ([]dto.PedidoDTO, int64, error)
	FindByID(ctx context.Context, id uint) (*dto.PedidoDTO, error)
	ListByCliente(ctx context.Context, clienteID uint, page, limit int) ([]dto.PedidoDTO, int64, error)
	AtualizarStatusPedido(ctx context.Context, id uint, status string) (*dto.PedidoDTO, error)
	Create(ctx context.Context, req *dto.CriarPedidoRequest) (*dto.PedidoDTO, error)
}

// ============================================
// CLIENTE SERVICE
// ============================================
