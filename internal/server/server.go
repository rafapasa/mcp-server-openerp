// internal/server/server.go - 100% DI - PRONTO PRO WIRE
package server

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/server/tools"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// MCPServer é a struct principal
type MCPServer struct {
	*server.MCPServer

	// Serviços - injetados
	cardapioService service.CardapioServiceInterface
	pedidoService service.PedidoServiceInterface
	carrinhoService service.CarrinhoServiceInterface

	// Clientes - injetados
	llm llm.LLMClient
	cache *redis.Client
	db *gorm.DB
}

// NewMCPServer - NÃO CRIA NADA DENTRO, SÓ RECEBE - WIRE READY
func NewMCPServer(
	db *gorm.DB,
	cache *redis.Client,
	llmClient llm.LLMClient,
	cardapioService service.CardapioServiceInterface,
	pedidoService service.PedidoServiceInterface,
	carrinhoService service.CarrinhoServiceInterface,
) *MCPServer {

	s := &MCPServer{
		MCPServer: server.NewMCPServer(
			"mcp-fastfood",
			"1.0.0",
			server.WithToolCapabilities(true),
		),
		db: db,
		cache: cache,
		llm: llmClient,
		cardapioService: cardapioService,
		pedidoService: pedidoService,
		carrinhoService: carrinhoService,
	}

	// Registra tools
	s.registerTools()

	return s
}

// registerTools registra todas as tools
func (s *MCPServer) registerTools() {
	deps := &tools.Dependencies{
		CardapioService: s.cardapioService,
		PedidoService: s.pedidoService,
		CarrinhoService: s.carrinhoService,
		LLMClient: s.llm,
	}
	tools.RegisterAllTools(s, deps)
}

// RegisterTool implementa a interface ToolRegistrar
func (s *MCPServer) RegisterTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	s.AddTool(tool, handler)
}
