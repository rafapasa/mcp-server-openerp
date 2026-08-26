// internal/server/tools/whatsapp.go - FINAL COM SEU MessageInput
package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

func RegisterWhatsAppTools(s ToolRegistrar, deps *Dependencies) {
	s.AddTool(whatsappTool(), whatsappHandler(deps))
}

func whatsappTool() mcp.Tool {
	return mcp.NewTool(
		"processar_mensagem_whatsapp",
		mcp.WithDescription("Recebe mensagem WhatsApp/MCP, detecta intenção via IA e processa carrinho - fluxo B2B/B2C unificado"),
		mcp.WithString("mensagem", mcp.Required(), mcp.Description("Texto do cliente")),
		mcp.WithString("tenant_id", mcp.Required()),
		mcp.WithString("cliente_id", mcp.Required()),
		mcp.WithString("cliente_nome"),
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
		tenantID, err := GetUintRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clienteID, err := GetUintRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		// cliente_nome não vai no MessageInput, mas pode ser usado em log/auditoria
		_, _ = GetString(args, "cliente_nome")

		logger.Info(ctx, "WhatsApp B2B tool -> ProcessarMensagem",
			zap.Uint("cliente_id", clienteID), zap.Uint("tenant_id", tenantID))

		// SUA STRUCT REAL
		input := dto.MessageInput{
			Text:   mensagem,
			Audio:  nil,
			Image:  nil,
			Source: models.SourceText, // ou models.MessageSourceTool - seu enum de B2B
		}

		resposta, err := deps.CarrinhoService.ProcessarMensagem(ctx, clienteID, tenantID, input)
		if err != nil {
			return handleLLMError(ctx, err, mensagem), nil
		}

		return mcp.NewToolResultText(resposta), nil
	}
}

func handleLLMError(ctx context.Context, err error, mensagem string) *mcp.CallToolResult {
	errMsg := strings.ToLower(err.Error())
	logger.Error(ctx, "Erro no ProcessarMensagem", zap.Error(err))
	if strings.Contains(errMsg, "503") || strings.Contains(errMsg, "unavailable") {
		return mcp.NewToolResultError("⚠ *IA indisponível.* Tente novamente.")
	}
	return mcp.NewToolResultError("❌ *Erro ao processar.* Tente novamente.")
}
