// internal/server/server.go
package server

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// MCPServer é a struct principal
type MCPServer struct {
	*server.MCPServer

	// Serviços
	cardapioService *service.CardapioService
	pedidoService   *service.PedidoService

	// Clientes
	llm   llm.LLMClient // Agora usa a interface
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

	s := &MCPServer{
		MCPServer: server.NewMCPServer(
			"mcp-fastfood",
			"1.0.0",
			server.WithToolCapabilities(true),
		),
		cardapioService: cardapioService,
		pedidoService:   pedidoService,
		llm:             llmClient,
		cache:           cache,
	}

	// Registra tools, resources e prompts
	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s
}
