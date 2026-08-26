package server

import (
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/server/tools"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

type MCPServer struct {
	mcp       *mcpserver.MCPServer
	sse       *mcpserver.SSEServer
	cardapio  service.CardapioServiceInterface
	pedido    service.PedidoServiceInterface
	carrinho  service.CarrinhoServiceInterface
	llmClient *llm.UnifiedLLM
	cfg       *config.Config
}

func NewMCPServer(
	cfg *config.Config,
	cardapioService service.CardapioServiceInterface,
	pedidoService service.PedidoServiceInterface,
	carrinhoService service.CarrinhoServiceInterface,
	llmClient *llm.UnifiedLLM,
) *MCPServer {
	mcpSrv := mcpserver.NewMCPServer(
		"mcp-openerp",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(false, false),
		mcpserver.WithRecovery(),
	)
	s := &MCPServer{
		mcp: mcpSrv, cardapio: cardapioService, pedido: pedidoService,
		carrinho: carrinhoService, llmClient: llmClient, cfg: cfg,
	}
	s.registerTools()
	s.sse = mcpserver.NewSSEServer(s.mcp, mcpserver.WithBaseURL("/mcp"))
	return s
}

func (s *MCPServer) registerTools() {
	deps := &tools.Dependencies{
		CardapioService: s.cardapio,
		PedidoService:   s.pedido,
		CarrinhoService: s.carrinho,
		LLMClient:       s.llmClient,
	}
	tools.RegisterAllTools(s.mcp, deps)
}

func (s *MCPServer) ServeStdio() error               { return mcpserver.ServeStdio(s.mcp) }
func (s *MCPServer) SSEServer() *mcpserver.SSEServer { return s.sse }
func (s *MCPServer) GetMCP() *mcpserver.MCPServer    { return s.mcp }
func (s *MCPServer) AddTool(tool mcp.Tool, handler mcpserver.ToolHandlerFunc) {
	s.mcp.AddTool(tool, handler)
}
