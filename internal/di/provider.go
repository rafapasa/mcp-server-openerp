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

// Providers que resolvem o []string
func provideGormDB(cfg *config.Config) *gorm.DB {
	return database.NewMySQL(cfg) // seu NewMySQL recebe *config.Config, não []string
}

func provideRedis(cfg *config.Config) (*database.Redis, error) {
	return database.NewRedis(cfg)
}

func provideRedisClient(r *database.Redis) *redis.Client {
	return r.Client
}

var providerSetObservability = wire.NewSet(
	health.NewHealthChecker,
)

// ... resto igual
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
)
var providerSetService = wire.NewSet(
	service.NewTenantService,
	service.NewAuthService,
	service.NewCardapioService,
	service.NewPedidoService,
	service.NewClienteService,
	service.NewCarrinhoService,
)
var providerSetHandlers = wire.NewSet(
	server.NewMCPServer,
	server.NewHttpServer,
	server.NewAPIHandlers,
	webhook.NewWhatsAppClient,
	webhook.NewWebhookHandler,
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
