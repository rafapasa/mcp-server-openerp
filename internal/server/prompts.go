package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *MCPServer) registerPrompts() {
	s.AddPrompt(mcp.NewPrompt("atendimento_pedido",
		mcp.WithPromptDescription("Template para atendimento de pedido"),
		mcp.WithArgument("cliente_nome", mcp.RequiredArgument(), mcp.ArgumentDescription("Nome do cliente")),
		mcp.WithArgument("pedido", mcp.RequiredArgument(), mcp.ArgumentDescription("Descrição do pedido")),
	), s.promptAtendimentoHandler())
}

func (s *MCPServer) promptAtendimentoHandler() func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		clienteNome := request.Params.Arguments["cliente_nome"]
		pedido := request.Params.Arguments["pedido"]

		return &mcp.GetPromptResult{
			Description: "Template de atendimento",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.NewTextContent(fmt.Sprintf("Cliente: %s\nPedido: %s\n\nPor favor, processe este pedido e confirme o valor total.",
						clienteNome, pedido,
					)),
				},
			},
		}, nil
	}
}
