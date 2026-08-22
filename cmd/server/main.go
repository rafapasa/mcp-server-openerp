package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/di"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Erro ao carregar config: %v", err)
	}

	zapLogger, cleanup, err := logger.NewLogger(cfg)
	if err != nil {
		log.Fatalf("Erro logger: %v", err)
	}
	defer cleanup()

	zapLogger.Info("Iniciando MCP Universal",
		zap.String("env", cfg.Environment),
		zap.String("log_level", cfg.LogLevel),
	)

	// WIRE - tudo injetado
	app, err := di.InitializeApp()
	if err != nil {
		zapLogger.Fatal("Erro ao inicializar app via Wire", zap.Error(err))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.WebhookPort
		if port == "" {
			port = "8080"
		}
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		zapLogger.Info("HttpServer ouvindo", zap.String("port", port))
		if err := app.Start(":" + port); err != nil {
			zapLogger.Fatal("Erro ao iniciar HttpServer", zap.Error(err))
		}
	}()

	<-ctx.Done()
	zapLogger.Info("Shutdown iniciado...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Shutdown(shutdownCtx); err != nil {
		zapLogger.Error("Erro no shutdown", zap.Error(err))
	}
	zapLogger.Info("Servidor finalizado")
}
