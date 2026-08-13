// cmd/stdio/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcp_server "github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
)

func main() {
	// ============================================
	// 1. CARREGA CONFIGURAÇÃO
	// ============================================
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	// ============================================
	// 2. INICIALIZA LOGGER
	// ============================================
	if err := logger.Init(logger.Config{
		Level:    cfg.LogLevel,
		Encoding: cfg.LogEncoding,
	}); err != nil {
		log.Fatalf("Erro ao inicializar logger: %v", err)
	}
	defer logger.Sync()

	zapLogger := logger.GetLogger()
	zapLogger.Info("Iniciando MCP Server (STDIO)",
		zap.String("log_level", cfg.LogLevel),
		zap.String("log_encoding", cfg.LogEncoding),
	)

	// ============================================
	// 3. CONECTA AO BANCO DE DADOS
	// ============================================
	db, err := database.NewMySQL(cfg, "")
	if err != nil {
		zapLogger.Fatal("Erro ao conectar ao banco de dados", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			zapLogger.Error("Erro ao fechar conexão com banco", zap.Error(err))
		}
	}()
	zapLogger.Info("Conectado ao banco de dados")

	// ============================================
	// 4. CONECTA AO REDIS
	// ============================================
	redisClient, err := database.NewRedis(cfg)
	if err != nil {
		zapLogger.Fatal("Erro ao conectar ao Redis", zap.Error(err))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			zapLogger.Error("Erro ao fechar conexão com Redis", zap.Error(err))
		}
	}()
	zapLogger.Info("Conectado ao Redis")

	// ============================================
	// 5. INICIALIZA CLIENTE LLM
	// ============================================
	llmClient, err := llm.NewLLMClientFromEnv()
	if err != nil {
		zapLogger.Fatal("Erro ao carregar LLM", zap.Error(err))
	}
	zapLogger.Info("LLM client inicializado",
		zap.String("provider", llmClient.GetProvider()),
		zap.String("model", llmClient.GetModel()),
	)

	// ============================================
	// 6. CRIA SERVIDOR MCP
	// ============================================
	mcpServer := server.NewMCPServer(db.DB, redisClient.Client, llmClient)
	zapLogger.Info("MCP Server criado com sucesso")

	// ============================================
	// 7. INICIA SERVIDOR (STDIO)
	// ============================================
	zapLogger.Info("Iniciando MCP Server (STDIO)...")
	zapLogger.Info("Aguardando conexões via STDIO")

	// STDIO não precisa de graceful shutdown tradicional,
	// mas podemos capturar sinais para encerrar o logger
	// e garantir que todos os recursos sejam liberados.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Captura sinais de encerramento
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		zapLogger.Info("Recebido sinal de encerramento, desligando...")
		cancel()
	}()

	// O servidor STDIO bloqueia a execução até ser finalizado
	// Usamos goroutine para não bloquear o shutdown
	errCh := make(chan error, 1)
	go func() {
		if err := mcp_server.ServeStdio(mcpServer.MCPServer); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	// Aguarda finalização ou erro
	select {
	case <-ctx.Done():
		zapLogger.Info("Servidor MCP finalizado")
	case err := <-errCh:
		if err != nil {
			zapLogger.Fatal("Erro ao iniciar servidor MCP", zap.Error(err))
		}
	}

	// ============================================
	// 8. SHUTDOWN GRACEFUL
	// ============================================
	zapLogger.Info("Desligando servidor MCP...")

	// Aguarda um curto período para operações pendentes
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()
	zapLogger.Info("Servidor MCP desligado com sucesso")
}
