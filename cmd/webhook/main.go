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
	llmClient, err := llm.NewLLMClient(cfg) // provider principal do .env (deepseek)
	if err != nil {
		zapLogger.Fatal("Erro ao carregar LLM", zap.Error(err))
	}

	// Especificos para media
	transcriber := media.NewGroqTranscriber(cfg)
	geminiLLM := llm.NewGeminiLLM(cfg)     // vision
	deepseekLLM := llm.NewDeepSeekLLM(cfg) // texto - pode ser mesmo que llmClient se provider=deepseek

	// Cria servidor webhook com todos os clients
	server := webhook.NewServer(
		db.GetDB(),
		redisClient.Client,
		llmClient,
		transcriber,
		geminiLLM,
		deepseekLLM,
	)

	port := cfg.WebhookPort
	if port == "" {
		port = "8080"
	}

	zapLogger.Info("Webhook server iniciado",
		zap.String("port", port),
		zap.String("provider_principal", llmClient.GetProvider()),
		zap.String("groq_model", cfg.LlmGroqModel),
		zap.String("gemini_model", cfg.LlmGeminiModel),
		zap.String("deepseek_model", cfg.LlmDeepSeekModel),
	)

	if err := server.Start(":" + port); err != nil {
		zapLogger.Fatal("Erro ao iniciar servidor", zap.Error(err))
	}
}
