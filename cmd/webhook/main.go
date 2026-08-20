package main

import (
	"log"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/media"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/webhook"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	if err := logger.Init(logger.Config{
		Level:    cfg.LogLevel,
		Encoding: cfg.LogEncoding,
	}); err != nil {
		log.Fatalf("Erro ao inicializar logger: %v", err)
	}
	defer logger.Sync()

	zapLogger := logger.GetLogger()
	zapLogger.Info("Iniciando servidor webhook",
		zap.String("log_level", cfg.LogLevel),
		zap.String("log_encoding", cfg.LogEncoding),
		zap.String("env", cfg.Environment),
	)

	db, err := database.NewMySQL(cfg, "")
	if err != nil {
		zapLogger.Fatal("Erro ao conectar ao banco", zap.Error(err))
	}
	defer db.Close()

	redisClient, err := database.NewRedis(cfg)
	if err != nil {
		zapLogger.Fatal("Erro ao conectar ao Redis", zap.Error(err))
	}

	// LLMs - 3 providers
	llmClient, err := llm.NewLLMClient(cfg)
	if err != nil {
		zapLogger.Fatal("Erro ao carregar LLM", zap.Error(err))
	}

	transcriber := media.NewGroqTranscriber(cfg)
	geminiLLM := llm.NewGeminiLLM(cfg)
	deepseekLLM := llm.NewDeepSeekLLM(cfg)

	server := webhook.NewServer(
		db.GetDB(),
		redisClient.Client,
		llmClient,
		transcriber,
		geminiLLM,
		deepseekLLM,
		cfg,
	)

	port := cfg.WebhookPort
	if port == "" {
		port = "8080"
	}

	zapLogger.Info("Webhook server iniciado",
		zap.String("port", port),
		zap.String("provider_principal", llmClient.GetProvider()),
	)

	if err := server.Start(":" + port); err != nil {
		zapLogger.Fatal("Erro ao iniciar servidor", zap.Error(err))
	}
}
