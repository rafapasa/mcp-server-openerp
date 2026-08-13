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
	cfg, _ := config.LoadConfig()

	// Inicializa logger
	if err := logger.Init(logger.Config{
		Level:    cfg.LogLevel,
		Encoding: cfg.LogEncoding,
	}); err != nil {
		log.Fatalf("Erro ao inicializar logger: %v", err)
	}
	defer logger.Sync()

	// Logger inicial
	zapLogger := logger.GetLogger()
	zapLogger.Info("Iniciando servidor webhook",
		zap.String("log_level", cfg.LogLevel),
		zap.String("log_encoding", cfg.LogEncoding),
	)

	// Conexão com banco de dados
	db, err := database.NewMySQL(cfg, "")
	if err != nil {
		zapLogger.Fatal("Erro ao conectar ao banco", zap.Error(err))
	}
	defer db.Close()

	// Conexão com Redis
	redisClient, err := database.NewRedis(cfg)
	if err != nil {
		zapLogger.Fatal("Erro ao conectar ao Redis", zap.Error(err))
	}

	// Cliente LLM
	llmClient, err := llm.NewLLMClientFromEnv()
	if err != nil {
		zapLogger.Fatal("Erro ao carregar LLM", zap.Error(err))
	}

	// Cria servidor webhook
	server := webhook.NewServer(db.GetDB(), redisClient.Client, llmClient)

	// Inicia servidor HTTP
	port := os.Getenv("WEBHOOK_PORT")
	if port == "" {
		port = "8080"
	}

	zapLogger.Info("Webhook server iniciado",
		zap.String("port", port),
	)

	if err := server.Start(":" + port); err != nil {
		zapLogger.Fatal("Erro ao iniciar servidor", zap.Error(err))
	}
}
