// internal/server/mcp_methods.go
package server

import (
	"context"
	"fmt"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/server/tools"
)

// GetCardapio busca o cardápio (wrapper para o service)
func (s *MCPServer) GetCardapio(tenantID string) ([]dto.ProdutoItem, error) {
	return s.cardapioService.GetCardapio(context.Background(), tenantID)
}

// ExtractIntent extrai a intenção do cliente
func (s *MCPServer) ExtractIntent(mensagem string, cardapio []dto.ProdutoItem) (*llm.IntencaoCliente, error) {
	return s.llm.ExtractIntent(mensagem, cardapio)
}

// AdicionarItemCarrinho adiciona um item ao carrinho
func (s *MCPServer) AdicionarItemCarrinho(ctx context.Context, clienteID, tenantID string, item dto.ItemCarrinho) error {
	return s.carrinhoService.AdicionarItem(ctx, clienteID, tenantID, item)
}

// RemoverItemCarrinho remove um item do carrinho
func (s *MCPServer) RemoverItemCarrinho(ctx context.Context, clienteID, tenantID string, nome string, quantidade int) error {
	return s.carrinhoService.RemoverItem(ctx, clienteID, tenantID, nome, quantidade)
}

// LimparCarrinho limpa o carrinho
func (s *MCPServer) LimparCarrinho(ctx context.Context, clienteID, tenantID string) error {
	return s.carrinhoService.LimparCarrinho(ctx, clienteID, tenantID)
}

// FinalizarCarrinho finaliza o pedido
func (s *MCPServer) FinalizarCarrinho(ctx context.Context, clienteID, tenantID, clienteNome string) (*dto.PedidoConfirmado, error) {
	return s.carrinhoService.FinalizarCarrinho(ctx, clienteID, tenantID, clienteNome)
}

// FormatarResumoCarrinho formata o resumo do carrinho
func (s *MCPServer) FormatarResumoCarrinho(ctx context.Context, clienteID, tenantID string) string {
	carrinho, err := s.carrinhoService.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao buscar carrinho: %v", err)
	}

	total := s.carrinhoService.CalcularTotal(carrinho)
	tempoEstimado := s.carrinhoService.CalcularTempoEstimado(carrinho)

	return tools.FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado)
}

// FormatarRespostaPedido formata a resposta do pedido
func (s *MCPServer) FormatarRespostaPedido(ctx context.Context, pedido *dto.PedidoConfirmado) string {
	return tools.FormatRespostaPedido(pedido)
}
