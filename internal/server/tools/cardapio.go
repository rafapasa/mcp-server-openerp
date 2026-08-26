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
)

// RegisterCardapioTools registra as tools de cardápio
func RegisterCardapioTools(s ToolRegistrar, deps *Dependencies) {
	s.AddTool(consultarCardapioTool(), consultarCardapioHandler(deps))
}

func consultarCardapioTool() mcp.Tool {
	return mcp.NewTool(
		"consultar_cardapio",
		mcp.WithDescription("Consulta o cardápio de um estabelecimento"),
		mcp.WithString(
			"tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
		mcp.WithString(
			"categoria",
			mcp.Description("Filtrar por categoria (ex: Lanches, Bebidas, Medicamentos, Higiene)"),
		),
		mcp.WithBoolean(
			"apenas_disponiveis",
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
		strTenantID, err := GetStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tenantID, err := strconv.Atoi(strTenantID)
		if err != nil {
			return mcp.NewToolResultError("ID do estabelecimento inválido"), nil
		}
		categoria, _ := GetString(args, "categoria")
		apenasDisponiveis := true
		if val, ok := args["apenas_disponiveis"].(bool); ok {
			apenasDisponiveis = val
		}

		cardapio, err := deps.CardapioService.GetCardapio(ctx, uint(tenantID))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}

		var filtrados []dto.ProdutoItem
		for _, item := range cardapio {
			if apenasDisponiveis && !item.Disponivel {
				continue
			}
			if categoria != "" && !strings.EqualFold(item.Categoria, categoria) {
				continue
			}
			filtrados = append(filtrados, item)
		}

		if len(filtrados) == 0 {
			return mcp.NewToolResultText("Nenhum item encontrado com os filtros"), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📋 Cardápio (%d itens):\n", len(filtrados)))
		sb.WriteString("Formato: ID - Nome - Preço - Status\n\n")

		for _, item := range filtrados {
			status := "✅"
			if !item.Disponivel {
				status = "❌"
			}
			// ID obrigatório pro fluxo novo de validação MySQL
			sb.WriteString(fmt.Sprintf("%s [%d] %s - R$ %.2f", status, item.ID, item.Nome, item.Preco))
			if item.Categoria != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", item.Categoria))
			}
			sb.WriteString("\n")
			if item.Descricao != "" {
				sb.WriteString(fmt.Sprintf("   %s\n", item.Descricao))
			}
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
}
