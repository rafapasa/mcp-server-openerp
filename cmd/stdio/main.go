// cmd/stdio/main.go
package main

import (
	"log"

	mcp_server "github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
)

func main() {
	// Carrega variáveis de ambiente
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
	llm := server.NewOpenAILLM()

	// Cria servidor MCP
	mcpServer := server.NewMCPServer(db.GetDB(), redisClient.GetClient(), llm)

	// Inicia servidor (STDIO)
	log.Println("Iniciando servidor MCP (STDIO)...")
	if err := mcp_server.ServeStdio(mcpServer.MCPServer); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
