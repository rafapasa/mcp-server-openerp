// internal/service/interfaces.go
package service

import "context"

// CardapioServiceInterface define as operações do serviço de cardápio
type CardapioServiceInterface interface {
	GetCardapio(ctx context.Context, tenantID string) ([]ProdutoItem, error)
	ItemExisteNoCardapio(cardapio []ProdutoItem, nome string) (bool, float64)
	EncontrarItemSimilar(cardapio []ProdutoItem, nome string) string
	FormatarCardapio(cardapio []ProdutoItem) string
}

// PedidoServiceInterface define as operações do serviço de pedidos
type PedidoServiceInterface interface {
	ProcessarPedido(ctx context.Context, tenantID, clienteID, clienteNome string, pedidoExtraido *PedidoExtraido) (*PedidoConfirmado, error)
}

// CarrinhoServiceInterface define as operações do serviço de carrinho
type CarrinhoServiceInterface interface {
	AdicionarItem(clienteID, tenantID string, item ItemCarrinho) error
	RemoverItem(clienteID, tenantID string, nome string, quantidade int) error
	GetCarrinho(clienteID, tenantID string) (*Carrinho, error)
	LimparCarrinho(clienteID, tenantID string) error
	FinalizarCarrinho(ctx context.Context, clienteID, tenantID, clienteNome string) (*PedidoConfirmado, error)
	CalcularTotal(carrinho *Carrinho) float64
	CalcularTempoEstimado(carrinho *Carrinho) int
}
