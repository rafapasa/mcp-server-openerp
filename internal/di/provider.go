// internal/di/provider.go
package di

import (
	"github.com/google/wire"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/health"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"github.com/rafapasa/mcp-server-openerp/internal/webhook"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var providerSetLLM = wire.NewSet(
	llm.NewUnifiedLLM,
)

func provideGormDB(cfg *config.Config) *gorm.DB {
	return database.NewMySQL(cfg)
}

func provideRedis(cfg *config.Config) (database.RedisInterface, error) {
	return database.NewRedis(cfg)
}

func provideRedisClient(r database.RedisInterface) *redis.Client {
	if r == nil {
		return nil
	}
	return r.GetClient()
}

var providerSetObservability = wire.NewSet(
	health.NewHealthChecker,
)

var providerSetConfig = wire.NewSet(
	config.LoadConfig,
)

var providerSetDataBase = wire.NewSet(
	provideGormDB,
	provideRedis,
	provideRedisClient,
)

var providerSetRepository = wire.NewSet(
	repository.NewTenantRepository,
	repository.NewProdutoRepository,
	repository.NewPedidoRepository,
	repository.NewClienteRepository,
	repository.NewEnderecoRepository,
	repository.NewUserRepository,
	repository.NewFormaPagamentoRepository,
	repository.NewPedidoPagamentoRepository,
)

var providerSetService = wire.NewSet(
	service.NewTenantService,
	service.NewAuthService,
	service.NewCardapioService,
	service.NewLLMService, // UnifiedLLM + CardapioServiceInterface
	service.NewPedidoService,
	service.NewClienteService,
	service.NewCarrinhoService, // agora recebe LLMServiceInterface
	service.NewFormaPagamentoService,
)

var providerSetHandlers = wire.NewSet(
	server.NewMCPServer,
	server.NewHttpServer,
	server.NewAPIHandlers,
	webhook.NewWhatsAppClient,
	webhook.NewWebhookHandler,
	webhook.NewProcessor,
)

var ProviderSet = wire.NewSet(
	providerSetConfig,
	providerSetObservability,
	providerSetDataBase,
	providerSetLLM,
	providerSetRepository,
	providerSetService,
	providerSetHandlers,
)
