// internal/service/interfaces.go
package service

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

// CardapioServiceInterface define as operações do serviço de cardápio
type CardapioServiceInterface interface {
	GetCardapio(ctx context.Context, tenantID uint) ([]dto.ProdutoItem, error)
	ItemExisteNoCardapio(cardapio []dto.ProdutoItem, nome string) (bool, float64)
	EncontrarItemSimilar(cardapio []dto.ProdutoItem, nome string) string
	FormatarCardapio(cardapio []dto.ProdutoItem) string
}

// PedidoServiceInterface define as operações do serviço de pedidos
type PedidoServiceInterface interface {
	ProcessarPedido(ctx context.Context, tenantID uint, clienteID uint, clienteNome string, pedidoExtraido *dto.PedidoExtraido) (*dto.PedidoConfirmado, error)
}

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
