// Package server contains the MCP server handlers and business logic.
package server

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// getCardapio busca o cardápio do restaurante
func (s *MCPServer) getCardapio(tenantID string) ([]service.ProdutoItem, error) {
	return s.cardapioService.GetCardapio(context.Background(), tenantID)
}

// itemExisteNoCardapio verifica se um item existe e retorna seu preço
func (s *MCPServer) itemExisteNoCardapio(cardapio []service.ProdutoItem, nome string) (bool, float64) {
	return s.cardapioService.ItemExisteNoCardapio(cardapio, nome)
}

// encontrarItemSimilar tenta encontrar um item similar no cardápio
func (s *MCPServer) encontrarItemSimilar(cardapio []service.ProdutoItem, nome string) string {
	return s.cardapioService.EncontrarItemSimilar(cardapio, nome)
}

// processarPedido salva o pedido no banco e retorna o pedido confirmado
func (s *MCPServer) processarPedido(
	tenantID, clienteID, clienteNome string,
	pedidoExtraido *service.PedidoExtraido,
) (*service.PedidoConfirmado, error) {
	return s.pedidoService.ProcessarPedido(
		context.Background(),
		tenantID,
		clienteID,
		clienteNome,
		pedidoExtraido,
	)
}

// formatarCardapio formata o cardápio para enviar no prompt da IA
func (s *MCPServer) formatarCardapio(cardapio []service.ProdutoItem) string {
	return s.cardapioService.FormatarCardapio(cardapio)
}
