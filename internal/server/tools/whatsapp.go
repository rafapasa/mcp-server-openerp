// internal/server/tools/whatsapp.go
package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// RegisterWhatsAppTools registra as tools relacionadas ao WhatsApp
func RegisterWhatsAppTools(s ToolRegistrar, deps *Dependencies) {
	s.RegisterTool(whatsappTool(), whatsappHandler(deps))
}

func whatsappTool() mcp.Tool {
	return mcp.NewTool("processar_mensagem_whatsapp",
		mcp.WithDescription("Recebe uma mensagem de WhatsApp, detecta a intenção (adicionar, remover, finalizar, visualizar) e processa"),
		mcp.WithString("mensagem",
			mcp.Required(),
			mcp.Description("Mensagem enviada pelo cliente no WhatsApp"),
		),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento (tenant)"),
		),
		mcp.WithString("cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente no WhatsApp (número de telefone)"),
		),
		mcp.WithString("cliente_nome",
			mcp.Description("Nome do cliente (opcional)"),
		),
	)
}

func whatsappHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		mensagem, _ := GetString(args, "mensagem")
		if mensagem == "" {
			return mcp.NewToolResultError("mensagem é obrigatória"), nil
		}

		tenantID, err := GetStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := GetStringRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteNome, _ := GetString(args, "cliente_nome")

		log.Printf("[WhatsApp] Processando mensagem de %s (%s): %s", clienteID, tenantID, mensagem)

		// Busca cardápio
		cardapio, err := deps.CardapioService.GetCardapio(ctx, tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}

		if len(cardapio) == 0 {
			return mcp.NewToolResultError("Cardápio não encontrado para este estabelecimento"), nil
		}

		// Detecta intenção do cliente
		intencao, err := deps.LLMClient.ExtractIntent(mensagem, cardapio)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao interpretar mensagem: %v", err)), nil
		}

		// Processa baseado na intenção
		switch intencao.Acao {
		case "adicionar", "add":
			return processarAdicionar(clienteID, tenantID, intencao, deps)
		case "remover", "remove":
			return processarRemover(clienteID, tenantID, intencao, deps)
		case "finalizar", "confirmar", "fechar":
			return processarFinalizar(ctx, clienteID, tenantID, clienteNome, deps)
		case "limpar", "clear":
			return processarLimpar(clienteID, tenantID, deps)
		default:
			// Visualizar carrinho
			return processarVisualizar(clienteID, tenantID, deps)
		}
	}
}

// processarAdicionar adiciona itens ao carrinho
func processarAdicionar(clienteID, tenantID string, intencao *llm.IntencaoCliente, deps *Dependencies) (*mcp.CallToolResult, error) {
	if len(intencao.Itens) == 0 {
		return mcp.NewToolResultError("Não foi possível identificar itens para adicionar ao carrinho"), nil
	}

	for _, item := range intencao.Itens {
		carrinhoItem := service.ItemCarrinho{
			Nome:       item.Nome,
			Quantidade: item.Quantidade,
			Observacao: item.Observacao,
			Preco:      item.PrecoUnitario,
		}
		if err := deps.CarrinhoService.AdicionarItem(clienteID, tenantID, carrinhoItem); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao adicionar item: %v", err)), nil
		}
	}

	// Busca carrinho atualizado
	carrinho, err := deps.CarrinhoService.GetCarrinho(clienteID, tenantID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
	}

	total := deps.CarrinhoService.CalcularTotal(carrinho)
	tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

	resposta := fmt.Sprintf("✅ Itens adicionados ao carrinho!\n\n%s",
		FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado))

	return mcp.NewToolResultText(resposta), nil
}

// processarRemover remove itens do carrinho
func processarRemover(clienteID, tenantID string, intencao *llm.IntencaoCliente, deps *Dependencies) (*mcp.CallToolResult, error) {
	if len(intencao.Itens) == 0 {
		return mcp.NewToolResultError("Não foi possível identificar itens para remover"), nil
	}

	for _, item := range intencao.Itens {
		if err := deps.CarrinhoService.RemoverItem(clienteID, tenantID, item.Nome, item.Quantidade); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao remover item: %v", err)), nil
		}
	}

	// Busca carrinho atualizado
	carrinho, err := deps.CarrinhoService.GetCarrinho(clienteID, tenantID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
	}

	total := deps.CarrinhoService.CalcularTotal(carrinho)
	tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

	resposta := fmt.Sprintf("✅ Itens removidos do carrinho!\n\n%s",
		FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado))

	return mcp.NewToolResultText(resposta), nil
}

// processarVisualizar mostra o carrinho atual
func processarVisualizar(clienteID, tenantID string, deps *Dependencies) (*mcp.CallToolResult, error) {
	carrinho, err := deps.CarrinhoService.GetCarrinho(clienteID, tenantID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
	}

	total := deps.CarrinhoService.CalcularTotal(carrinho)
	tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

	resposta := FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado)
	return mcp.NewToolResultText(resposta), nil
}

// processarFinalizar finaliza o pedido
func processarFinalizar(ctx context.Context, clienteID, tenantID, clienteNome string, deps *Dependencies) (*mcp.CallToolResult, error) {
	pedidoConfirmado, err := deps.CarrinhoService.FinalizarCarrinho(ctx, clienteID, tenantID, clienteNome)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Erro ao finalizar pedido: %v", err)), nil
	}

	resposta := FormatRespostaPedido(pedidoConfirmado)
	return mcp.NewToolResultText(resposta), nil
}

// processarLimpar limpa o carrinho
func processarLimpar(clienteID, tenantID string, deps *Dependencies) (*mcp.CallToolResult, error) {
	if err := deps.CarrinhoService.LimparCarrinho(clienteID, tenantID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Erro ao limpar carrinho: %v", err)), nil
	}

	return mcp.NewToolResultText("🗑️ Carrinho limpo com sucesso!\n\nAdicione novos itens usando: *quero um X-Bacon*"), nil
}
