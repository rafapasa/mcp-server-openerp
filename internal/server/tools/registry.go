// internal/server/tools/registry.go
package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// ToolRegistrar interface para registrar tools
type ToolRegistrar interface {
	RegisterTool(tool mcp.Tool, handler server.ToolHandlerFunc)
}

// Dependencies contém todas as dependências necessárias para as tools
type Dependencies struct {
	CardapioService service.CardapioServiceInterface
	PedidoService   service.PedidoServiceInterface
	CarrinhoService service.CarrinhoServiceInterface
	LLMClient       llm.LLMClient
}

// RegisterAllTools registra todas as tools do servidor
func RegisterAllTools(s ToolRegistrar, deps *Dependencies) {
	// Registra tools do WhatsApp
	RegisterWhatsAppTools(s, deps)

	// Registra tools do Carrinho
	RegisterCarrinhoTools(s, deps)

	// Registra tools de Pedido
	RegisterPedidoTools(s, deps)

	// Registra tools de Cardápio
	RegisterCardapioTools(s, deps)
}