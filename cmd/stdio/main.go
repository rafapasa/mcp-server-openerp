package main

import (
	"log"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/di"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	_, cleanup, err := logger.NewLogger(cfg)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer cleanup()

	mcpServer, err := di.InitializeMCPServer()
	if err != nil {
		log.Fatalf("wire mcp: %v", err)
	}

	if err := mcpServer.ServeStdio(); err != nil {
		log.Fatalf("stdio: %v", err)
	}
}
