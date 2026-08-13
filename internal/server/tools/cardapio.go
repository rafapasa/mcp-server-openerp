// internal/server/tools/cardapio.go
package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

// RegisterCardapioTools registra as tools de cardápio
func RegisterCardapioTools(s ToolRegistrar, deps *Dependencies) {
	s.RegisterTool(consultarCardapioTool(), consultarCardapioHandler(deps))
}

func consultarCardapioTool() mcp.Tool {
	return mcp.NewTool("consultar_cardapio",
		mcp.WithDescription("Consulta o cardápio de um estabelecimento"),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
		mcp.WithString("categoria",
			mcp.Description("Filtrar por categoria (ex: Lanches, Bebidas, Medicamentos, Higiene)"),
		),
		mcp.WithBoolean("apenas_disponiveis",
			mcp.Description("Mostrar apenas itens disponíveis"),
			mcp.DefaultBool(true),
		),
	)
}

func consultarCardapioHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			logger.Warn(ctx, "Erro ao extrair argumentos", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		strTenantID, err := GetStringRequired(args, "tenant_id")
		if err != nil {
			logger.Warn(ctx, "tenant_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := strconv.Atoi(strTenantID)
		if err != nil {
			logger.Warn(ctx, "tenant_id não é um número válido",
				zap.String("tenant_id", strTenantID),
				zap.Error(err),
			)
			return mcp.NewToolResultError("ID do estabelecimento inválido"), nil
		}

		categoria, _ := GetString(args, "categoria")
		apenasDisponiveis := true
		if val, ok := args["apenas_disponiveis"].(bool); ok {
			apenasDisponiveis = val
		}

		logger.Debug(ctx, "Consultando cardápio",
			zap.Int("tenant_id", tenantID),
			zap.String("categoria", categoria),
			zap.Bool("apenas_disponiveis", apenasDisponiveis),
		)

		cardapio, err := deps.CardapioService.GetCardapio(ctx, uint(tenantID))
		if err != nil {
			logger.Error(ctx, "Erro ao buscar cardápio",
				zap.Error(err),
				zap.Int("tenant_id", tenantID),
			)
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}

		if len(cardapio) == 0 {
			logger.Warn(ctx, "Cardápio vazio", zap.Int("tenant_id", tenantID))
			return mcp.NewToolResultText("Nenhum item encontrado no cardápio"), nil
		}

		var filtrados []dto.ProdutoItem
		for _, item := range cardapio {
			if apenasDisponiveis && !item.Disponivel {
				continue
			}
			if categoria != "" && item.Categoria != categoria {
				continue
			}
			filtrados = append(filtrados, item)
		}

		if len(filtrados) == 0 {
			logger.Debug(ctx, "Nenhum item encontrado com os filtros aplicados",
				zap.Int("tenant_id", tenantID),
				zap.String("categoria", categoria),
				zap.Bool("apenas_disponiveis", apenasDisponiveis),
			)
			return mcp.NewToolResultText("Nenhum item encontrado no cardápio com os filtros selecionados"), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📋 Cardápio (%d itens):\n\n", len(filtrados)))

		for _, item := range filtrados {
			status := "✅"
			if !item.Disponivel {
				status = "❌"
			}
			sb.WriteString(fmt.Sprintf("%s **%s** - R$ %.2f\n", status, item.Nome, item.Preco))
			if item.Descricao != "" {
				sb.WriteString(fmt.Sprintf("   %s\n", item.Descricao))
			}
			sb.WriteString("\n")
		}

		logger.Info(ctx, "Cardápio consultado com sucesso",
			zap.Int("tenant_id", tenantID),
			zap.Int("itens_encontrados", len(filtrados)),
		)

		return mcp.NewToolResultText(sb.String()), nil
	}
}
