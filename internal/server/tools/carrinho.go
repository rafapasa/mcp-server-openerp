// internal/server/tools/carrinho.go
package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

// RegisterCarrinhoTools registra as tools de gerenciamento de carrinho
func RegisterCarrinhoTools(s ToolRegistrar, deps *Dependencies) {
	s.RegisterTool(adicionarAoCarrinhoTool(), adicionarAoCarrinhoHandler(deps))
	s.RegisterTool(removerDoCarrinhoTool(), removerDoCarrinhoHandler(deps))
	s.RegisterTool(visualizarCarrinhoTool(), visualizarCarrinhoHandler(deps))
	s.RegisterTool(finalizarPedidoTool(), finalizarPedidoHandler(deps))
	s.RegisterTool(limparCarrinhoTool(), limparCarrinhoHandler(deps))
}

// ============================================
// adicionar_ao_carrinho
// ============================================

func adicionarAoCarrinhoTool() mcp.Tool {
	return mcp.NewTool("adicionar_ao_carrinho",
		mcp.WithDescription("Adiciona um ou mais itens ao carrinho do cliente"),
		mcp.WithString("cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
		mcp.WithString("item_nome",
			mcp.Required(),
			mcp.Description("Nome do item a ser adicionado"),
		),
		mcp.WithNumber("quantidade",
			mcp.Description("Quantidade do item (padrão: 1)"),
			mcp.DefaultNumber(1),
		),
		mcp.WithString("observacao",
			mcp.Description("Observações sobre o item (ex: sem cebola)"),
		),
	)
}

func adicionarAoCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetStringRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		itemNome, err := GetStringRequired(args, "item_nome")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		quantidade := 1
		if qtd, ok := args["quantidade"].(float64); ok && qtd > 0 {
			quantidade = int(qtd)
		}

		observacao, _ := GetString(args, "observacao")

		// Busca cardápio para validar e obter preço
		cardapio, err := deps.CardapioService.GetCardapio(ctx, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}

		existe, preco := deps.CardapioService.ItemExisteNoCardapio(cardapio, itemNome)
		if !existe {
			similar := deps.CardapioService.EncontrarItemSimilar(cardapio, itemNome)
			if similar != "" {
				_, preco = deps.CardapioService.ItemExisteNoCardapio(cardapio, similar)
				itemNome = similar
			} else {
				return mcp.NewToolResultError(fmt.Sprintf("Item '%s' não encontrado no cardápio", itemNome)), nil
			}
		}

		item := dto.ItemCarrinho{
			Nome:       itemNome,
			Quantidade: quantidade,
			Observacao: observacao,
			Preco:      preco,
		}

		if err := deps.CarrinhoService.AdicionarItem(ctx, clienteID, tenantID, item); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao adicionar item: %v", err)), nil
		}

		carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
		}

		total := deps.CarrinhoService.CalcularTotal(carrinho)
		tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

		resposta := fmt.Sprintf("✅ Adicionado: %dx **%s** ao carrinho!\n\n%s",
			quantidade, itemNome, FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado))

		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// remover_do_carrinho
// ============================================

func removerDoCarrinhoTool() mcp.Tool {
	return mcp.NewTool("remover_do_carrinho",
		mcp.WithDescription("Remove um item do carrinho do cliente"),
		mcp.WithString("cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
		mcp.WithString("item_nome",
			mcp.Required(),
			mcp.Description("Nome do item a ser removido"),
		),
		mcp.WithNumber("quantidade",
			mcp.Description("Quantidade a remover (padrão: 1, remova 0 para remover todos)"),
			mcp.DefaultNumber(1),
		),
	)
}

func removerDoCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetStringRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		itemNome, err := GetStringRequired(args, "item_nome")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		quantidade := 1
		if qtd, ok := args["quantidade"].(float64); ok && qtd > 0 {
			quantidade = int(qtd)
		}

		if err := deps.CarrinhoService.RemoverItem(ctx, clienteID, tenantID, itemNome, quantidade); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao remover item: %v", err)), nil
		}

		carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
		}

		total := deps.CarrinhoService.CalcularTotal(carrinho)
		tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

		resposta := fmt.Sprintf("✅ Removido: %dx **%s** do carrinho!\n\n%s",
			quantidade, itemNome, FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado))

		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// visualizar_carrinho
// ============================================

func visualizarCarrinhoTool() mcp.Tool {
	return mcp.NewTool("visualizar_carrinho",
		mcp.WithDescription("Visualiza o carrinho atual do cliente com itens, total e tempo estimado"),
		mcp.WithString("cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
	)
}

func visualizarCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetStringRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
		}

		total := deps.CarrinhoService.CalcularTotal(carrinho)
		tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

		resposta := FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado)
		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// finalizar_pedido
// ============================================

func finalizarPedidoTool() mcp.Tool {
	return mcp.NewTool("finalizar_pedido",
		mcp.WithDescription("Finaliza o pedido convertendo o carrinho em um pedido confirmado"),
		mcp.WithString("cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
		mcp.WithString("cliente_nome",
			mcp.Description("Nome do cliente (opcional)"),
		),
	)
}

func finalizarPedidoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetStringRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteNome, _ := GetString(args, "cliente_nome")

		pedidoConfirmado, err := deps.CarrinhoService.FinalizarCarrinho(ctx, clienteID, tenantID, clienteNome)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao finalizar pedido: %v", err)), nil
		}

		resposta := FormatRespostaPedido(pedidoConfirmado)
		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// limpar_carrinho
// ============================================

func limparCarrinhoTool() mcp.Tool {
	return mcp.NewTool("limpar_carrinho",
		mcp.WithDescription("Limpa todo o carrinho do cliente"),
		mcp.WithString("cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
	)
}

func limparCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetStringRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if err := deps.CarrinhoService.LimparCarrinho(ctx, clienteID, tenantID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao limpar carrinho: %v", err)), nil
		}

		return mcp.NewToolResultText("🗑️ Carrinho limpo com sucesso!\n\nAdicione novos itens usando: *quero um X-Bacon*"), nil
	}
}
