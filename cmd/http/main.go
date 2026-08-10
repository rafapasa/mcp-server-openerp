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
	"github.com/rafapasa/mcp-server-openerp/internal/server"
)

func main() {
	cfg := config.LoadConfig()

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
	mcpServer := server.NewMCPServer(db.GetDB(), redisClient.GetClient(), llmClient)

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
