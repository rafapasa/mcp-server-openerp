// internal/server/tools/pedido.go
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
			logger.Warn(ctx, "Erro ao extrair argumentos", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			logger.Warn(ctx, "tenant_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			logger.Warn(ctx, "cliente_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		itens, err := GetItems(args)
		if err != nil {
			logger.Warn(ctx, "Erro ao extrair itens", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteNome, _ := GetString(args, "cliente_nome")
		observacoes, _ := GetString(args, "observacoes")

		logger.Info(ctx, "Processando pedido via dashboard",
			zap.Uint("tenant_id", tenantID),
			zap.Uint("cliente_id", clienteID),
			zap.String("cliente_nome", clienteNome),
			zap.Int("itens_count", len(itens)),
		)

		pedidoExtraido := &dto.PedidoExtraido{
			Itens:       itens,
			Observacoes: observacoes,
		}

		pedidoConfirmado, err := deps.PedidoService.ProcessarPedido(ctx, tenantID, clienteID, clienteNome, pedidoExtraido)
		if err != nil {
			logger.Error(ctx, "Erro ao processar pedido",
				zap.Error(err),
				zap.Uint("tenant_id", tenantID),
				zap.Uint("cliente_id", clienteID),
			)
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao processar pedido: %v", err)), nil
		}

		logger.Info(ctx, "Pedido processado com sucesso",
			zap.Int("pedido_id", pedidoConfirmado.ID),
			zap.Float64("total", pedidoConfirmado.Total),
			zap.String("status", pedidoConfirmado.Status),
		)

		return mcp.NewToolResultText(
			fmt.Sprintf("✅ Pedido #%d processado com sucesso!\nTotal: R$ %.2f",
				pedidoConfirmado.ID, pedidoConfirmado.Total),
		), nil
	}
}
