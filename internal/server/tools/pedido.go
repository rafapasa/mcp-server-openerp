// internal/server/tools/pedido.go
package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

// RegisterPedidoTools registra as tools de pedido
func RegisterPedidoTools(s ToolRegistrar, deps *Dependencies) {
	s.RegisterTool(processarPedidoTool(), processarPedidoHandler(deps))
}

func processarPedidoTool() mcp.Tool {
	return mcp.NewTool("processar_pedido_restaurante",
		mcp.WithDescription("Processa um pedido manualmente (usado pelo dashboard do estabelecimento)"),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
		mcp.WithString("cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente"),
		),
		mcp.WithString("cliente_nome",
			mcp.Description("Nome do cliente"),
		),
		mcp.WithArray("itens",
			mcp.Required(),
			mcp.Description("Lista de itens do pedido"),
		),
		mcp.WithString("observacoes",
			mcp.Description("Observações gerais do pedido"),
		),
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

		itens, err := GetItems(args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteNome, _ := GetString(args, "cliente_nome")
		observacoes, _ := GetString(args, "observacoes")

		pedidoExtraido := &dto.PedidoExtraido{
			Itens:       itens,
			Observacoes: observacoes,
		}

		pedidoConfirmado, err := deps.PedidoService.ProcessarPedido(ctx, tenantID, clienteID, clienteNome, pedidoExtraido)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao processar pedido: %v", err)), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("✅ Pedido #%d processado com sucesso!\nTotal: R$ %.2f",
				pedidoConfirmado.ID, pedidoConfirmado.Total),
		), nil
	}
}
