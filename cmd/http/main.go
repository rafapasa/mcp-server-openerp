// cmd/http/main.go
package main

import (
	"log"
	"net/http"
	"os"

	mcp_server "github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

func main() {
	cfg, _ := config.LoadConfig()

	// Conexão com banco de dados
	db, err := database.NewMySQL(cfg, "")
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco: %v", err)
	}
	defer db.Close()

	// Conexão com Redis
	redisClient, err := database.NewRedis(cfg)
	if err != nil {
		log.Fatalf("Erro ao conectar ao Redis: %v", err)
	}

	// Cliente OpenAI
	llmClient, err := llm.NewLLMClientFromEnv()
	if err != nil {
		log.Fatalf("Erro ao carregar LLM Client: %v", err)
	}

	// Cria servidor MCP
	tenantRepo := repository.NewTenantRepository(db.DB)
	produtoRepo := repository.NewProdutoRepository(db.DB)
	pedidoRepo := repository.NewPedidoRepository(db.DB)

	cardapioService := service.NewCardapioService(produtoRepo, tenantRepo, redisClient.Client)
	pedidoService := service.NewPedidoService(pedidoRepo, cardapioService)

	carrinhoService := service.NewCarrinhoService(redisClient.Client, cardapioService, pedidoService, produtoRepo, llmClient)

	mcpServer := server.NewMCPServer(db.DB, redisClient.Client, llmClient, cardapioService, pedidoService, carrinhoService)

	// Configura HTTP Server
	s := mcp_server.NewStreamableHTTPServer(mcpServer.MCPServer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Iniciando servidor HTTP na porta %s...", port)
	if err := http.ListenAndServe(":"+port, s); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
