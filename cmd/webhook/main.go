package main

import (
	"log"
	"os"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/webhook"
	"go.uber.org/zap"
)

func main() {
	// Carrega configuração
	cfg := config.LoadConfig()

	// Logger inicial
	zapLogger := logger.GetLogger()
	zapLogger.Info("Iniciando servidor webhook",
		zap.String("log_level", cfg.LogLevel),
		zap.String("log_encoding", cfg.LogEncoding),
	)

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

	// Cliente LLM
	llmClient, err := llm.NewLLMClientFromEnv()
	if err != nil {
		log.Fatalf("Erro ao carregar LLM: %v", err)
	}

	// Cria servidor webhook
	server := webhook.NewServer(db.DB, redisClient.Client, llmClient)

	// Inicia servidor HTTP
	port := os.Getenv("WEBHOOK_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Webhook server iniciado na porta %s", port)
	log.Fatal(server.Start(":" + port))
}
