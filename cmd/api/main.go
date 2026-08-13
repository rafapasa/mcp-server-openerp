// cmd/api/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/api"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

func main() {
	// ============================================
	// 1. CARREGA CONFIGURAÇÃO
	// ============================================
	cfg, _ := config.LoadConfig()

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
	zapLogger.Info("Iniciando API Server",
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
	defer db.Close()

	zapLogger.Info("Conectado ao banco de dados")

	// ============================================
	// 4. CONECTA AO REDIS
	// ============================================
	redisClient, err := database.NewRedis(cfg)
	if err != nil {
		zapLogger.Warn("Erro ao conectar ao Redis (continuando sem cache)", zap.Error(err))
	} else {
		zapLogger.Info("Conectado ao Redis")
		defer redisClient.Close()
	}

	// ============================================
	// 5. CRIA SERVIDOR API
	// ============================================
	apiServer := api.NewServer(db.GetDB(), redisClient.Client)

	// ============================================
	// 6. DEFINE PORTA
	// ============================================
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8081"
	}
	addr := ":" + port

	// ============================================
	// 7. INICIA SERVIDOR EM GOROUTINE
	// ============================================
	go func() {
		zapLogger.Info("API Server iniciado",
			zap.String("addr", addr),
		)
		if err := apiServer.Start(addr); err != nil {
			zapLogger.Fatal("Erro ao iniciar API Server", zap.Error(err))
		}
	}()

	// ============================================
	// 8. AGUARDA SINAL DE ENCERRAMENTO
	// ============================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Recebido sinal de encerramento, desligando API Server...")

	// ============================================
	// 9. SHUTDOWN GRACEFUL
	// ============================================
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		zapLogger.Error("Erro ao desligar API Server", zap.Error(err))
	} else {
		zapLogger.Info("API Server desligado com sucesso")
	}
}
