// internal/server/server.go
package server

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/rafapasa/mcp-server-openerp/internal/server/tools"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// MCPServer é a struct principal
type MCPServer struct {
	*server.MCPServer

	// Serviços
	cardapioService service.CardapioServiceInterface
	pedidoService   service.PedidoServiceInterface
	carrinhoService service.CarrinhoServiceInterface

	// Clientes
	llm   llm.LLMClient
	cache *redis.Client
}

// NewMCPServer cria uma nova instância do servidor
func NewMCPServer(db *gorm.DB, cache *redis.Client, llmClient llm.LLMClient) *MCPServer {
	// Inicializa repositórios
	tenantRepo := repository.NewTenantRepository(db)
	produtoRepo := repository.NewProdutoRepository(db)
	pedidoRepo := repository.NewPedidoRepository(db)

	// Inicializa serviços
	cardapioService := service.NewCardapioService(produtoRepo, tenantRepo, cache)
	pedidoService := service.NewPedidoService(pedidoRepo, cardapioService)
	carrinhoService := service.NewCarrinhoService(cache, cardapioService, pedidoService)

	s := &MCPServer{
		MCPServer: server.NewMCPServer(
			"mcp-fastfood",
			"1.0.0",
			server.WithToolCapabilities(true),
		),
		cardapioService: cardapioService,
		pedidoService:   pedidoService,
		carrinhoService: carrinhoService,
		llm:             llmClient,
		cache:           cache,
	}

	// Registra tools usando o novo sistema
	s.registerTools()

	return s
}

// registerTools registra todas as tools usando o novo sistema
func (s *MCPServer) registerTools() {
	// Cria dependências para as tools
	deps := &tools.Dependencies{
		CardapioService: s.cardapioService,
		PedidoService:   s.pedidoService,
		CarrinhoService: s.carrinhoService,
		LLMClient:       s.llm,
	}

	// Registra todas as tools
	tools.RegisterAllTools(s, deps)
}

// RegisterTool implementa a interface ToolRegistrar
func (s *MCPServer) RegisterTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	s.AddTool(tool, handler)
}
