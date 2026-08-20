// internal/webhook/routes.go - FINAL 100% FIBER - CORRIGIDO PRE-DEPLOY
package webhook

import (
	"context"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rafapasa/mcp-server-openerp/internal/cache"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/media"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/health"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/metrics"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/middleware"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/tracing"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Server struct {
	app         *fiber.App
	handler     *WebhookHandler
	db          *gorm.DB
	cache       *redis.Client
	cacheLayer  *cache.Cache // ADICIONADO: cache wrapper real
	llmClient   llm.LLMClient
	transcriber *media.GroqTranscriber
	geminiLLM   llm.LLMClient
	deepseekLLM llm.LLMClient
	healthCheck *health.HealthChecker
	cfg         *config.Config
}

func NewServer(db *gorm.DB,
	cacheClient *redis.Client,
	llmClient llm.LLMClient,
	transcriber *media.GroqTranscriber,
	geminiLLM llm.LLMClient,
	deepseekLLM llm.LLMClient,
	cfg *config.Config) *Server {

	tenantRepo := repository.NewTenantRepository(db)
	produtoRepo := repository.NewProdutoRepository(db)
	pedidoRepo := repository.NewPedidoRepository(db)

	cardapioService := service.NewCardapioService(produtoRepo, tenantRepo, cacheClient)
	pedidoService := service.NewPedidoService(pedidoRepo, cardapioService)
	clienteRepo := repository.NewClienteRepository(db)
	cliEndRepo := repository.NewEnderecoRepository(db)
	clienteService := service.NewClienteService(clienteRepo, cliEndRepo)

	carrinhoService := service.NewCarrinhoService(cacheClient, cardapioService, pedidoService, produtoRepo, llmClient)

	// Cache Layer real com GetOrSet
	cacheLayer := cache.New(cacheClient)

	mcpServer := server.NewMCPServer(db, cacheClient, llmClient, cardapioService, pedidoService, carrinhoService)
	// Se NewWhatsAppClient precisa de cfg, passa: NewWhatsAppClient(cfg)
	whatsApp := NewWhatsAppClient()
	handler := NewWebhookHandler(mcpServer, whatsApp, clienteService, transcriber, geminiLLM, deepseekLLM, cfg)
	handler.SetCache(cacheLayer) // FIX 1: Injeta cache

	hc := health.NewDefaultHealthChecker(db, cacheClient)

	return &Server{
		handler:     handler,
		db:          db,
		cache:       cacheClient,
		cacheLayer:  cacheLayer,
		llmClient:   llmClient,
		transcriber: transcriber,
		geminiLLM:   geminiLLM,
		deepseekLLM: deepseekLLM,
		healthCheck: hc,
		cfg:         cfg,
	}
}

func (s *Server) Start(addr string) error {
	metrics.Init()
	tracing.Init(tracing.Config{
		Enabled:      s.cfg.TracingEnabled,
		Endpoint:     s.cfg.TracingEndpoint,
		ServiceName:  s.cfg.TracingServiceName,
		SamplingRate: s.cfg.TracingSamplingRate,
	})

	rateLimiter := middleware.NewRateLimiterFromEnv()
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rateLimiter.Cleanup()
		}
	}()

	app := fiber.New(fiber.Config{
		AppName:      "MCP Webhook - Fiber 100%",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middleware.SecurityHeadersFiber(s.cfg))
	app.Use(middleware.TracingFiber())
	app.Use(middleware.MetricsFiber())
	app.Use(middleware.LoggerFiber())
	app.Use(middleware.RateLimitFiber(rateLimiter))

	s.app = app

	// Health & Metrics
	app.Get("/health", adaptor.HTTPHandlerFunc(health.HealthHandler(s.healthCheck)))
	app.Get("/ready", adaptor.HTTPHandlerFunc(health.ReadinessHandler(s.healthCheck)))
	app.Get("/status", adaptor.HTTPHandlerFunc(health.StatusHandler(s.healthCheck)))
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Webhook 100% Fiber
	app.Get("/webhook", s.handler.HandleVerifyWebhookFiber)
	app.Post("/webhook", s.handler.HandleWebhookFiber)

	logger.GetLogger().Info("Webhook Fiber 100% - SEM sanitize.go, SEM Mux",
		zap.String("addr", addr),
		zap.Bool("cache_enabled", s.cacheLayer != nil),
	)
	return app.Listen(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
