// internal/server/tools/whatsapp.go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/helpers"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

// RegisterWhatsAppTools registra as tools relacionadas ao WhatsApp
func RegisterWhatsAppTools(s ToolRegistrar, deps *Dependencies) {
	s.AddTool(whatsappTool(), whatsappHandler(deps))
}

func whatsappTool() mcp.Tool {
	return mcp.NewTool(
		"processar_mensagem_whatsapp",
		mcp.WithDescription("Recebe uma mensagem de WhatsApp, detecta a intenção (adicionar, remover, finalizar, visualizar) e processa"),
		mcp.WithString(
			"mensagem",
			mcp.Required(),
			mcp.Description("Mensagem enviada pelo cliente no WhatsApp"),
		),
		mcp.WithString(
			"tenant_id",
			mcp.Required(),
			mcp.Description("ID do estabelecimento (tenant)"),
		),
		mcp.WithString(
			"cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente no WhatsApp (número de telefone)"),
		),
		mcp.WithString(
			"cliente_nome",
			mcp.Description("Nome do cliente (opcional)"),
		),
	)
}

func whatsappHandler(deps *Dependencies) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := GetArguments(request)
		if err != nil {
			logger.Warn(ctx, "Erro ao extrair argumentos", zap.Error(err))
			return mcp.NewToolResultError(err.Error()), nil
		}

		mensagem, _ := GetString(args, "mensagem")
		if mensagem == "" {
			return mcp.NewToolResultError("mensagem é obrigatória"), nil
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

		logger.Info(
			ctx, "Processando mensagem de WhatsApp via tool",
			zap.Uint("cliente_id", clienteID),
			zap.Uint("tenant_id", tenantID),
			zap.String("mensagem", mensagem),
		)

		// Busca cardápio
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
			logger.Warn(ctx, "Cardápio vazio para tenant", zap.Uint("tenant_id", tenantID))
			return mcp.NewToolResultError("Cardápio não encontrado para este estabelecimento"), nil
		}

		// ✅ Detecta intenção do cliente (com retry já implementado no LLM)
		intencao, err := deps.LLMClient.ExtractIntent(ctx, mensagem, cardapio)
		if err != nil {
			// ✅ Trata erro da LLM
			return handleLLMError(ctx, err, mensagem), nil
		}

		// Processa baseado na intenção
		switch intencao.Acao {
		case "adicionar", "add":
			return processarAdicionar(ctx, clienteID, tenantID, intencao, deps)
		case "remover", "remove":
			return processarRemover(ctx, clienteID, tenantID, intencao, deps)
		case "finalizar", "confirmar", "fechar":
			return processarFinalizar(ctx, clienteID, tenantID, clienteNome, deps)
		case "limpar", "clear":
			return processarLimpar(ctx, clienteID, tenantID, deps)
		default:
			// Visualizar carrinho
			return processarVisualizar(ctx, clienteID, tenantID, deps)
		}
	}
}

// ============================================
// ✅ HANDLE LLM ERROR
// ============================================

// handleLLMError trata erros da LLM e retorna mensagem amigável
func handleLLMError(ctx context.Context, err error, mensagem string) *mcp.CallToolResult {
	logger.Error(
		ctx, "Erro ao processar mensagem com LLM",
		zap.Error(err),
		zap.String("mensagem", mensagem),
	)

	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)

	// ✅ Verifica se é erro de indisponibilidade (503, UNAVAILABLE, high demand)
	if strings.Contains(errMsgLower, "503") ||
		strings.Contains(errMsgLower, "unavailable") ||
		strings.Contains(errMsgLower, "high demand") ||
		strings.Contains(errMsgLower, "temporarily indisponível") {

		logger.Warn(
			ctx, "LLM indisponível, retornando mensagem amigável",
			zap.Error(err),
		)

		return mcp.NewToolResultError(
			"⚠️ *Desculpe, estou com dificuldades técnicas no momento.*\n\n" +
				"O serviço de IA está temporariamente indisponível.\n" +
				"Por favor, tente novamente em alguns segundos.\n\n" +
				"Se o problema persistir, entre em contato com o suporte.",
		)
	}

	// ✅ Verifica se é erro de timeout
	if strings.Contains(errMsgLower, "timeout") ||
		strings.Contains(errMsgLower, "deadline") {

		logger.Warn(
			ctx, "LLM timeout, retornando mensagem amigável",
			zap.Error(err),
		)

		return mcp.NewToolResultError(
			"⏱️ *A requisição demorou muito para ser processada.*\n\n" +
				"Por favor, tente novamente com uma mensagem mais curta ou simplificada.",
		)
	}

	// ✅ Verifica se é erro de autenticação
	if strings.Contains(errMsgLower, "unauthorized") ||
		strings.Contains(errMsgLower, "invalid api key") ||
		strings.Contains(errMsgLower, "403") {

		logger.Error(
			ctx, "Erro de autenticação com LLM",
			zap.Error(err),
		)

		return mcp.NewToolResultError(
			"🔒 *Erro de configuração no serviço de IA.*\n\n" +
				"Por favor, entre em contato com o suporte para resolver o problema.",
		)
	}

	// ✅ Erro genérico
	return mcp.NewToolResultError(
		"❌ *Ops! Algo deu errado ao processar sua mensagem.*\n\n" +
			"Por favor, tente novamente em alguns instantes.\n" +
			"Se o problema persistir, entre em contato com o suporte.",
	)
}

// ============================================
// PROCESSADORES DE AÇÃO
// ============================================

// processarAdicionar adiciona itens ao carrinho
func processarAdicionar(ctx context.Context, clienteID, tenantID uint, intencao *dto.IntencaoCliente, deps *Dependencies) (*mcp.CallToolResult, error) {
	if len(intencao.Itens) == 0 {
		return mcp.NewToolResultError("Não foi possível identificar itens para adicionar ao carrinho"), nil
	}

	logger.Info(
		ctx, "Adicionando itens ao carrinho",
		zap.Uint("cliente_id", clienteID),
		zap.Int("itens_count", len(intencao.Itens)),
	)

	for _, item := range intencao.Itens {
		carrinhoItem := dto.ItemCarrinho{
			ProdutoItem: item.ProdutoItem,
			Quantidade:  item.Quantidade,
			Observacao:  item.Observacao,
			Preco:       item.PrecoUnitario,
		}
		if err := deps.CarrinhoService.AdicionarItem(ctx, clienteID, tenantID, carrinhoItem); err != nil {
			logger.Error(
				ctx, "Erro ao adicionar item ao carrinho",
				zap.Error(err),
				zap.String("item_nome", item.ProdutoItem.Nome),
			)
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao adicionar item: %v", err)), nil
		}
	}

	// Busca carrinho atualizado
	carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		logger.Error(ctx, "Erro ao buscar carrinho", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
	}

	total := deps.CarrinhoService.CalcularTotal(carrinho)
	tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

	resposta := fmt.Sprintf("✅ Itens adicionados ao carrinho!\n\n%s",
		helpers.FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado))

	return mcp.NewToolResultText(resposta), nil
}

// processarRemover remove itens do carrinho
func processarRemover(ctx context.Context, clienteID, tenantID uint, intencao *dto.IntencaoCliente, deps *Dependencies) (*mcp.CallToolResult, error) {
	if len(intencao.Itens) == 0 {
		return mcp.NewToolResultError("Não foi possível identificar itens para remover"), nil
	}

	logger.Info(
		ctx, "Removendo itens do carrinho",
		zap.Uint("cliente_id", clienteID),
		zap.Int("itens_count", len(intencao.Itens)),
	)

	for _, item := range intencao.Itens {
		if err := deps.CarrinhoService.RemoverItem(ctx, clienteID, tenantID, item.ProdutoItem.Nome, item.Quantidade); err != nil {
			logger.Error(
				ctx, "Erro ao remover item do carrinho",
				zap.Error(err),
				zap.String("item_nome", item.ProdutoItem.Nome),
			)
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao remover item: %v", err)), nil
		}
	}

	// Busca carrinho atualizado
	carrinho, err := deps.CarrinhoService.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		logger.Error(ctx, "Erro ao buscar carrinho", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar carrinho: %v", err)), nil
	}

	total := deps.CarrinhoService.CalcularTotal(carrinho)
	tempoEstimado := deps.CarrinhoService.CalcularTempoEstimado(carrinho)

	resposta := fmt.Sprintf("✅ Itens removidos do carrinho!\n\n%s",
		helpers.FormatResumoCarrinho(carrinho.Itens, total, tempoEstimado))

	return mcp.NewToolResultText(resposta), nil
}

// processarVisualizar mostra o carrinho atual
func processarVisualizar(ctx context.Context, clienteID, tenantID uint, deps *Dependencies) (*mcp.CallToolResult, error) {
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

// processarFinalizar finaliza o pedido
func processarFinalizar(ctx context.Context, clienteID, tenantID uint, clienteNome string, deps *Dependencies) (*mcp.CallToolResult, error) {
	logger.Info(
		ctx, "Finalizando pedido",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("tenant_id", tenantID),
	)

	pedidoConfirmado, err := deps.CarrinhoService.FinalizarCarrinho(ctx, clienteID, tenantID, clienteNome)
	if err != nil {
		logger.Error(ctx, "Erro ao finalizar pedido", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf("Erro ao finalizar pedido: %v", err)), nil
	}

	resposta := helpers.FormatRespostaPedido(pedidoConfirmado)
	return mcp.NewToolResultText(resposta), nil
}

// processarLimpar limpa o carrinho
func processarLimpar(ctx context.Context, clienteID, tenantID uint, deps *Dependencies) (*mcp.CallToolResult, error) {
	logger.Info(
		ctx, "Limpando carrinho",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("tenant_id", tenantID),
	)

	if err := deps.CarrinhoService.LimparCarrinho(ctx, clienteID, tenantID); err != nil {
		logger.Error(ctx, "Erro ao limpar carrinho", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf("Erro ao limpar carrinho: %v", err)), nil
	}

	return mcp.NewToolResultText("🗑️ Carrinho limpo com sucesso!\n\nAdicione novos itens usando: *quero um X-Bacon*"), nil
}
