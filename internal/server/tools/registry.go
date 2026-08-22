// internal/server/tools/registry.go - CORRIGIDO
package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// ToolRegistrar - interface compatível com *mcp-go/server.MCPServer
// O mcp-go usa AddTool, não RegisterTool
type ToolRegistrar interface {
	AddTool(tool mcp.Tool, handler server.ToolHandlerFunc)
}

// Dependencies contém todas as dependências necessárias para as tools
type Dependencies struct {
	CardapioService service.CardapioServiceInterface
	PedidoService   service.PedidoServiceInterface
	CarrinhoService service.CarrinhoServiceInterface
	LLMClient       *llm.UnifiedLLM // <- corrigido, você usa TextLLM, não LLMClient genérico
}

// RegisterAllTools registra todas as tools do servidor
func RegisterAllTools(s ToolRegistrar, deps *Dependencies) {
	RegisterWhatsAppTools(s, deps)
	RegisterCarrinhoTools(s, deps)
	RegisterPedidoTools(s, deps)
	RegisterCardapioTools(s, deps)
}
