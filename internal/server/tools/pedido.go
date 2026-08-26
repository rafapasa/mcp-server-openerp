// internal/server/tools/pedido.go - COMPLETO - usando dto existente
package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

func RegisterPedidoTools(s ToolRegistrar, deps *Dependencies) {
	s.AddTool(processarPedidoTool(), processarPedidoHandler(deps))
}

func processarPedidoTool() mcp.Tool {
	return mcp.NewTool(
		"processar_pedido_restaurante",
		mcp.WithDescription("Processa pedido manual. Cada item DEVE ter produto_id (ID MySQL obtido via consultar_cardapio). Validação segura por ID."),
		mcp.WithString("tenant_id", mcp.Required()),
		mcp.WithString("cliente_id", mcp.Required()),
		mcp.WithString("cliente_nome"),
		mcp.WithArray("itens", mcp.Required(), mcp.Description(`[{"produto_id": 123, "quantidade": 1, "observacao": "sem cebola"}] - produto_id obrigatório`)),
		mcp.WithString("observacoes"),
	)
}

func processarPedidoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clienteNome, _ := GetString(args, "cliente_nome")
		observacoes, _ := GetString(args, "observacoes")

		rawItens, ok := args["itens"]
		if !ok {
			return mcp.NewToolResultError("itens é obrigatório"), nil
		}
		itensSlice, ok := rawItens.([]interface{})
		if !ok || len(itensSlice) == 0 {
			return mcp.NewToolResultError("itens deve ser array não vazio"), nil
		}

		// Carrega cardápio 1x - evita N+1
		cardapio, err := deps.CardapioService.GetCardapio(ctx, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}

		var itensValidados []dto.ItemPedidoInput
		for idx, raw := range itensSlice {
			m, ok := raw.(map[string]interface{})
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("item[%d] inválido", idx)), nil
			}
			prodIDRaw, ok := m["produto_id"]
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("item[%d] sem produto_id - use consultar_cardapio", idx)), nil
			}
			prodIDFloat, ok := prodIDRaw.(float64)
			if !ok || prodIDFloat <= 0 {
				return mcp.NewToolResultError(fmt.Sprintf("item[%d] produto_id inválido", idx)), nil
			}
			produtoID := uint(prodIDFloat)

			produto, err := deps.CardapioService.BuscarProdutoPorIdNoCardapio(cardapio, produtoID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("item[%d]: %v", idx, err)), nil
			}
			if !produto.Disponivel {
				return mcp.NewToolResultError(fmt.Sprintf("item[%d] [%d] %s indisponível", idx, produto.ID, produto.Nome)), nil
			}

			qtd := 1
			if q, ok := m["quantidade"].(float64); ok && q > 0 {
				qtd = int(q)
			}
			obs := ""
			if o, ok := m["observacao"].(string); ok {
				obs = o
			}

			// Usa seu DTO existente
			itensValidados = append(itensValidados, dto.ItemPedidoInput{
				ProdutoItem:   *produto,
				Quantidade:    qtd,
				Observacao:    obs,
				PrecoUnitario: produto.Preco,
			})
		}

		logger.Info(ctx, "Processando pedido B2B por ID", zap.Uint("tenant_id", tenantID), zap.Int("itens", len(itensValidados)))

		pedidoExtraido := &dto.PedidoExtraido{
			Itens:       itensValidados,
			Observacoes: observacoes,
		}

		pedidoConfirmado, err := deps.PedidoService.ProcessarPedido(ctx, tenantID, clienteID, clienteNome, pedidoExtraido)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao processar: %v", err)), nil
		}

		mensagem := fmt.Sprintf("✅ Pedido #%d | Total R$ %.2f | %d itens validados por ID MySQL",
			pedidoConfirmado.ID, pedidoConfirmado.Total, len(itensValidados))
		return mcp.NewToolResultText(mensagem), nil
	}
}
