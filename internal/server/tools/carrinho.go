// internal/server/tools/carrinho.go
package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/helpers"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

// RegisterCarrinhoTools registra as tools de gerenciamento de carrinho
func RegisterCarrinhoTools(s ToolRegistrar, deps *Dependencies) {
	s.AddTool(adicionarAoCarrinhoTool(), adicionarAoCarrinhoHandler(deps))
	s.AddTool(removerDoCarrinhoTool(), removerDoCarrinhoHandler(deps))
	s.AddTool(visualizarCarrinhoTool(), visualizarCarrinhoHandler(deps))
	s.AddTool(finalizarPedidoTool(), finalizarPedidoHandler(deps))
	s.AddTool(limparCarrinhoTool(), limparCarrinhoHandler(deps))
}

// ============================================
// adicionar_ao_carrinho
// ============================================

func adicionarAoCarrinhoTool() mcp.Tool {
	return mcp.NewTool(
		"adicionar_ao_carrinho",
		mcp.WithDescription("Adiciona um ou mais itens ao carrinho do cliente"),
		mcp.WithString(
			"cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString(
			"tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
		mcp.WithString(
			"item_nome",
			mcp.Required(),
			mcp.Description("Nome do item a ser adicionado"),
		),
		mcp.WithNumber(
			"quantidade",
			mcp.Description("Quantidade do item (padrão: 1)"),
			mcp.DefaultNumber(1),
		),
		mcp.WithString(
			"observacao",
			mcp.Description("Observações sobre o item (ex: sem cebola)"),
		),
	)
}

func adicionarAoCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			logger.Warn(ctx, "Erro ao extrair argumentos", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			logger.Warn(ctx, "cliente_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			logger.Warn(ctx, "tenant_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		itemNome, err := GetStringRequired(args, "item_nome")
		if err != nil {
			logger.Warn(ctx, "item_nome inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		quantidade := 1
		if qtd, ok := args["quantidade"].(float64); ok && qtd > 0 {
			quantidade = int(qtd)
		}

		observacao, _ := GetString(args, "observacao")

		logger.Info(
			ctx, "Adicionando item ao carrinho",
			zap.Uint("cliente_id", clienteID),
			zap.Uint("tenant_id", tenantID),
			zap.String("item_nome", itemNome),
			zap.Int("quantidade", quantidade),
		)

		// Busca cardápio para validar e obter preço
		cardapio, err := deps.CardapioService.GetCardapio(ctx, tenantID)
		if err != nil {
			logger.Error(
				ctx, "Erro ao buscar cardápio",
				zap.Error(err),
				zap.Uint("tenant_id", tenantID),
			)
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}

		if len(cardapio) == 0 {
			logger.Warn(ctx, "Cardápio vazio", zap.Uint("tenant_id", tenantID))
			return mcp.NewToolResultError("Cardápio não encontrado para este estabelecimento"), nil
		}

		produtoItem, err := deps.CardapioService.ItemExisteNoCardapio(cardapio, itemNome)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if produtoItem == nil {
			similar := deps.CardapioService.EncontrarItemSimilar(cardapio, itemNome)
			if similar != "" {
				produtoItem, err = deps.CardapioService.ItemExisteNoCardapio(cardapio, similar)
				logger.Info(
					ctx, "Item corrigido",
					zap.String("original", itemNome),
					zap.String("corrigido", similar),
				)
				itemNome = similar
			} else {
				logger.Warn(
					ctx, "Item não encontrado no cardápio",
					zap.String("item_nome", itemNome),
					zap.Uint("tenant_id", tenantID),
				)
				return mcp.NewToolResultError(fmt.Sprintf("Item '%s' não encontrado no cardápio", itemNome)), nil
			}
		}

		item := dto.ItemCarrinho{
			ProdutoItem: *produtoItem,
			Quantidade:  quantidade,
			Observacao:  observacao,
			Preco:       *&produtoItem.Preco,
		}

		if err := deps.CarrinhoService.AdicionarItem(ctx, clienteID, tenantID, item); err != nil {
			logger.Error(
				ctx, "Erro ao adicionar item ao carrinho",
				zap.Error(err),
				zap.String("item_nome", itemNome),
				zap.Uint("cliente_id", clienteID),
			)
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao adicionar item: %v", err)), nil
		}

		carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			logger.Error(ctx, "Erro ao buscar carrinho", zap.Error(err))
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
		}

		total := deps.CarrinhoService.CalcularTotal(carrinho)
		tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

		logger.Info(
			ctx, "Item adicionado com sucesso",
			zap.String("item_nome", itemNome),
			zap.Int("total_itens", len(carrinho.Itens)),
			zap.Float64("total", total),
		)

		resposta := fmt.Sprintf("✅ Adicionado: %dx **%s** ao carrinho!\n\n%s",
			quantidade, itemNome, helpers.FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado))

		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// remover_do_carrinho
// ============================================

func removerDoCarrinhoTool() mcp.Tool {
	return mcp.NewTool(
		"remover_do_carrinho",
		mcp.WithDescription("Remove um item do carrinho do cliente"),
		mcp.WithString(
			"cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString(
			"tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
		mcp.WithString(
			"item_nome",
			mcp.Required(),
			mcp.Description("Nome do item a ser removido"),
		),
		mcp.WithNumber(
			"quantidade",
			mcp.Description("Quantidade a remover (padrão: 1, remova 0 para remover todos)"),
			mcp.DefaultNumber(1),
		),
	)
}

func removerDoCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			logger.Warn(ctx, "Erro ao extrair argumentos", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			logger.Warn(ctx, "cliente_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			logger.Warn(ctx, "tenant_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		itemNome, err := GetStringRequired(args, "item_nome")
		if err != nil {
			logger.Warn(ctx, "item_nome inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		quantidade := 1
		if qtd, ok := args["quantidade"].(float64); ok && qtd > 0 {
			quantidade = int(qtd)
		}

		logger.Info(
			ctx, "Removendo item do carrinho",
			zap.Uint("cliente_id", clienteID),
			zap.Uint("tenant_id", tenantID),
			zap.String("item_nome", itemNome),
			zap.Int("quantidade", quantidade),
		)

		if err := deps.CarrinhoService.RemoverItem(ctx, clienteID, tenantID, itemNome, quantidade); err != nil {
			logger.Error(
				ctx, "Erro ao remover item do carrinho",
				zap.Error(err),
				zap.String("item_nome", itemNome),
				zap.Uint("cliente_id", clienteID),
			)
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao remover item: %v", err)), nil
		}

		carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			logger.Error(ctx, "Erro ao buscar carrinho", zap.Error(err))
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
		}

		total := deps.CarrinhoService.CalcularTotal(carrinho)
		tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

		logger.Info(
			ctx, "Item removido com sucesso",
			zap.String("item_nome", itemNome),
			zap.Int("total_itens", len(carrinho.Itens)),
		)

		resposta := fmt.Sprintf("✅ Removido: %dx **%s** do carrinho!\n\n%s",
			quantidade, itemNome, helpers.FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado))

		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// visualizar_carrinho
// ============================================

func visualizarCarrinhoTool() mcp.Tool {
	return mcp.NewTool(
		"visualizar_carrinho",
		mcp.WithDescription("Visualiza o carrinho atual do cliente com itens, total e tempo estimado"),
		mcp.WithString(
			"cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString(
			"tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
	)
}

func visualizarCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			logger.Warn(ctx, "Erro ao extrair argumentos", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			logger.Warn(ctx, "cliente_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			logger.Warn(ctx, "tenant_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		logger.Debug(
			ctx, "Visualizando carrinho",
			zap.Uint("cliente_id", clienteID),
			zap.Uint("tenant_id", tenantID),
		)

		carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			logger.Error(ctx, "Erro ao buscar carrinho", zap.Error(err))
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
		}

		total := deps.CarrinhoService.CalcularTotal(carrinho)
		tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

		resposta := helpers.FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado)
		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// finalizar_pedido
// ============================================

func finalizarPedidoTool() mcp.Tool {
	return mcp.NewTool(
		"finalizar_pedido",
		mcp.WithDescription("Finaliza o pedido convertendo o carrinho em um pedido confirmado"),
		mcp.WithString(
			"cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString(
			"tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
		mcp.WithString(
			"cliente_nome",
			mcp.Description("Nome do cliente (opcional)"),
		),
	)
}

func finalizarPedidoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			logger.Warn(ctx, "Erro ao extrair argumentos", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			logger.Warn(ctx, "cliente_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			logger.Warn(ctx, "tenant_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteNome, _ := GetString(args, "cliente_nome")

		logger.Info(
			ctx, "Finalizando pedido",
			zap.Uint("cliente_id", clienteID),
			zap.Uint("tenant_id", tenantID),
		)

		pedidoConfirmado, err := deps.CarrinhoService.FinalizarCarrinho(ctx, clienteID, tenantID, clienteNome)
		if err != nil {
			logger.Error(
				ctx, "Erro ao finalizar pedido",
				zap.Error(err),
				zap.Uint("cliente_id", clienteID),
			)
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao finalizar pedido: %v", err)), nil
		}

		logger.Info(
			ctx, "Pedido finalizado com sucesso",
			zap.Int("pedido_id", pedidoConfirmado.ID),
			zap.Float64("total", pedidoConfirmado.Total),
		)

		resposta := helpers.FormatRespostaPedido(pedidoConfirmado)
		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// limpar_carrinho
// ============================================

func limparCarrinhoTool() mcp.Tool {
	return mcp.NewTool(
		"limpar_carrinho",
		mcp.WithDescription("Limpa todo o carrinho do cliente"),
		mcp.WithString(
			"cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente (número de telefone)"),
		),
		mcp.WithString(
			"tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento"),
		),
	)
}

func limparCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			logger.Warn(ctx, "Erro ao extrair argumentos", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			logger.Warn(ctx, "cliente_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			logger.Warn(ctx, "tenant_id inválido", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		logger.Info(
			ctx, "Limpando carrinho",
			zap.Uint("cliente_id", clienteID),
			zap.Uint("tenant_id", tenantID),
		)

		if err := deps.CarrinhoService.LimparCarrinho(ctx, clienteID, tenantID); err != nil {
			logger.Error(
				ctx, "Erro ao limpar carrinho",
				zap.Error(err),
				zap.Uint("cliente_id", clienteID),
			)
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao limpar carrinho: %v", err)), nil
		}

		return mcp.NewToolResultText("🗑️ Carrinho limpo com sucesso!\n\nAdicione novos itens usando: *quero um X-Bacon*"), nil
	}
}
