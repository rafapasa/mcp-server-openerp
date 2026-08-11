// internal/server/tools/cardapio.go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
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
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		categoria, _ := GetString(args, "categoria")
		apenasDisponiveis := true
		if val, ok := args["apenas_disponiveis"].(bool); ok {
			apenasDisponiveis = val
		}

		cardapio, err := deps.CardapioService.GetCardapio(ctx, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}

		var filtrados []service.ProdutoItem
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
			return mcp.NewToolResultText("Nenhum item encontrado no cardápio"), nil
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

		return mcp.NewToolResultText(sb.String()), nil
	}
}
