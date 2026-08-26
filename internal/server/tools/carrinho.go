// internal/server/tools/carrinho.go - COMPLETO - FLUXO NOVO ID MySQL
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

func RegisterCarrinhoTools(s ToolRegistrar, deps *Dependencies) {
	s.AddTool(adicionarAoCarrinhoTool(), adicionarAoCarrinhoHandler(deps))
	s.AddTool(removerDoCarrinhoTool(), removerDoCarrinhoHandler(deps))
	s.AddTool(visualizarCarrinhoTool(), visualizarCarrinhoHandler(deps))
	s.AddTool(finalizarPedidoTool(), finalizarPedidoHandler(deps))
	s.AddTool(limparCarrinhoTool(), limparCarrinhoHandler(deps))
}

// ============================================
// adicionar_ao_carrinho - ID MySQL
// ============================================
func adicionarAoCarrinhoTool() mcp.Tool {
	return mcp.NewTool(
		"adicionar_ao_carrinho",
		mcp.WithDescription("Adiciona item ao carrinho usando ID do cardápio. Use consultar_cardapio para obter o ID."),
		mcp.WithString("cliente_id", mcp.Required(), mcp.Description("ID do cliente")),
		mcp.WithString("tenant_id", mcp.Required(), mcp.Description("ID do estabelecimento")),
		mcp.WithNumber("produto_id", mcp.Required(), mcp.Description("ID do produto no MySQL - obtido via consultar_cardapio")),
		mcp.WithNumber("quantidade", mcp.Description("Quantidade"), mcp.DefaultNumber(1)),
		mcp.WithString("observacao", mcp.Description("Observações ex: sem cebola")),
	)
}

func adicionarAoCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		produtoID, err := GetUintRequired(args, "produto_id")
		if err != nil {
			return mcp.NewToolResultError("produto_id é obrigatório - use consultar_cardapio para obter o ID"), nil
		}
		quantidade := 1
		if qtd, ok := args["quantidade"].(float64); ok && qtd > 0 {
			quantidade = int(qtd)
		}
		observacao, _ := GetString(args, "observacao")

		logger.Info(ctx, "Adicionando item ao carrinho B2B",
			zap.Uint("cliente_id", clienteID), zap.Uint("tenant_id", tenantID),
			zap.Uint("produto_id", produtoID), zap.Int("quantidade", quantidade))

		cardapio, err := deps.CardapioService.GetCardapio(ctx, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}
		if len(cardapio) == 0 {
			return mcp.NewToolResultError("Cardápio vazio para este estabelecimento"), nil
		}

		produtoItem, err := deps.CardapioService.BuscarProdutoPorIdNoCardapio(cardapio, produtoID)
		if err != nil {
			return nil, err
		}
		if produtoItem == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Produto ID %d não existe no tenant %d", produtoID, tenantID)), nil
		}
		if !produtoItem.Disponivel {
			return mcp.NewToolResultError(fmt.Sprintf("Produto [%d] %s indisponível", produtoItem.ID, produtoItem.Nome)), nil
		}

		item := dto.ItemCarrinho{
			ProdutoItem: *produtoItem,
			Quantidade:  quantidade,
			Observacao:  observacao,
			Preco:       produtoItem.Preco,
		}

		if err := deps.CarrinhoService.AdicionarItem(ctx, clienteID, tenantID, item); err != nil {
			logger.Error(ctx, "Erro ao adicionar item", zap.Error(err))
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao adicionar: %v", err)), nil
		}

		carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
		}
		total := deps.CarrinhoService.CalcularTotal(carrinho)
		tempo := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

		resposta := fmt.Sprintf("✅ Adicionado: %dx [%d] %s ao carrinho!\n\n%s",
			quantidade, produtoItem.ID, produtoItem.Nome, helpers.FormatResumoCarrinho(carrinho.Itens, total, tempo))
		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// remover_do_carrinho - ID MySQL
// ============================================
func removerDoCarrinhoTool() mcp.Tool {
	return mcp.NewTool(
		"remover_do_carrinho",
		mcp.WithDescription("Remove item do carrinho por ID do produto"),
		mcp.WithString("cliente_id", mcp.Required()),
		mcp.WithString("tenant_id", mcp.Required()),
		mcp.WithNumber("produto_id", mcp.Required(), mcp.Description("ID do produto")),
		mcp.WithNumber("quantidade", mcp.Description("Qtd a remover, 0 remove todos"), mcp.DefaultNumber(1)),
	)
}

func removerDoCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		produtoID, err := GetUintRequired(args, "produto_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		quantidade := 1
		if qtd, ok := args["quantidade"].(float64); ok {
			quantidade = int(qtd)
		}

		cardapio, err := deps.CardapioService.GetCardapio(ctx, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}
		produto, err := deps.CardapioService.BuscarProdutoPorIdNoCardapio(cardapio, produtoID)
		if err != nil {
			return nil, err
		}
		if produto == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Produto ID %d não existe no tenant %d", produtoID, tenantID)), nil
		}
		itemCarrinho := dto.ItemCarrinho{
			ProdutoItem: *produto,
			Quantidade:  quantidade,
		}

		if err := deps.CarrinhoService.RemoverItem(ctx, clienteID, tenantID, itemCarrinho, quantidade); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao remover: %v", err)), nil
		}
		carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
		}
		total := deps.CarrinhoService.CalcularTotal(carrinho)
		tempo := deps.CarrinhoService.CalcularTempoEstimado(carrinho)
		resposta := fmt.Sprintf("✅ Removido!\n\n%s", helpers.FormatResumoCarrinho(carrinho.Itens, total, tempo))
		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// visualizar_carrinho
// ============================================
func visualizarCarrinhoTool() mcp.Tool {
	return mcp.NewTool(
		"visualizar_carrinho",
		mcp.WithDescription("Visualiza carrinho atual com total e tempo estimado"),
		mcp.WithString("cliente_id", mcp.Required()),
		mcp.WithString("tenant_id", mcp.Required()),
	)
}

func visualizarCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
		}
		total := deps.CarrinhoService.CalcularTotal(carrinho)
		tempo := deps.CarrinhoService.CalcularTempoEstimado(carrinho)
		return mcp.NewToolResultText(helpers.FormatResumoCarrinho(carrinho.Itens, total, tempo)), nil
	}
}

// ============================================
// finalizar_pedido
// ============================================
func finalizarPedidoTool() mcp.Tool {
	return mcp.NewTool(
		"finalizar_pedido",
		mcp.WithDescription("Finaliza pedido convertendo carrinho em pedido confirmado"),
		mcp.WithString("cliente_id", mcp.Required()),
		mcp.WithString("tenant_id", mcp.Required()),
		mcp.WithString("cliente_nome", mcp.Description("Nome do cliente opcional")),
	)
}

func finalizarPedidoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clienteNome, _ := GetString(args, "cliente_nome")

		pedido, err := deps.CarrinhoService.FinalizarCarrinho(ctx, clienteID, tenantID, clienteNome)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao finalizar: %v", err)), nil
		}
		return mcp.NewToolResultText(helpers.FormatRespostaPedido(pedido)), nil
	}
}

// ============================================
// limpar_carrinho
// ============================================
func limparCarrinhoTool() mcp.Tool {
	return mcp.NewTool(
		"limpar_carrinho",
		mcp.WithDescription("Limpa todo o carrinho do cliente"),
		mcp.WithString("cliente_id", mcp.Required()),
		mcp.WithString("tenant_id", mcp.Required()),
	)
}

func limparCarrinhoHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := deps.CarrinhoService.LimparCarrinho(ctx, clienteID, tenantID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao limpar: %v", err)), nil
		}
		return mcp.NewToolResultText("🗑 Carrinho limpo com sucesso!"), nil
	}
}
