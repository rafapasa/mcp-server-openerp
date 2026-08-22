//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
)

func InitializeApp() (*server.HttpServer, error) {
	wire.Build(ProviderSet)
	return nil, nil
}

func InitializeMCPServer() (*server.MCPServer, error) {
	wire.Build(
		providerSetConfig,
		providerSetDataBase,
		providerSetLLM,
		providerSetRepository,
		providerSetService,
		providerSetHandlers, // só MCPServer
	)
	return nil, nil
}
