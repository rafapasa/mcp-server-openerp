// internal/webhook/routes.go - FINAL 100% FIBER - SEM MUX, SEM SANITIZE, SEM ADAPTOR NO WEBHOOK
package webhook

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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
	llmClient   llm.LLMClient
	transcriber *media.GroqTranscriber
	geminiLLM   llm.LLMClient
	deepseekLLM llm.LLMClient
	healthCheck *health.HealthChecker
	cfg         *config.Config
}

func NewServer(db *gorm.DB,
	cache *redis.Client,
	llmClient llm.LLMClient,
	transcriber *media.GroqTranscriber,
	geminiLLM llm.LLMClient,
	deepseekLLM llm.LLMClient,
	cfg *config.Config) *Server {

	tenantRepo := repository.NewTenantRepository(db)
	produtoRepo := repository.NewProdutoRepository(db)
	pedidoRepo := repository.NewPedidoRepository(db)

	cardapioService := service.NewCardapioService(produtoRepo, tenantRepo, cache)
	pedidoService := service.NewPedidoService(pedidoRepo, cardapioService)
	clienteRepo := repository.NewClienteRepository(db)
	cliEndRepo := repository.NewEnderecoRepository(db)
	clienteService := service.NewClienteService(clienteRepo, cliEndRepo)

	_ = service.NewCarrinhoService(cache, cardapioService, pedidoService, produtoRepo, llmClient)

	mcpServer := server.NewMCPServer(db, cache, llmClient)
	whatsApp := NewWhatsAppClient()
	handler := NewWebhookHandler(mcpServer, whatsApp, clienteService, transcriber, geminiLLM, deepseekLLM, cfg)
	hc := health.NewDefaultHealthChecker(db, cache)

	return &Server{
		handler:     handler,
		db:          db,
		cache:       cache,
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
		IdleTimeout:  60 * time.Second,
	})

	app.Use(recover.New())
	app.Use(cors.New())

	// Security Headers only - NO SANITIZE
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		if config.IsProduction() {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		return c.Next()
	})

	app.Use(func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/health") || c.Path() == "/metrics" {
			return c.Next()
		}
		key := c.Get("X-Tenant-ID", c.IP())
		if ok, _ := rateLimiter.Allow(key); !ok {
			return c.Status(429).JSON(fiber.Map{"error": "rate limit"})
		}
		return c.Next()
	})

	s.app = app

	// Health & Metrics
	app.Get("/health", adaptor.HTTPHandlerFunc(health.HealthHandler(s.healthCheck)))
	app.Get("/ready", adaptor.HTTPHandlerFunc(health.ReadinessHandler(s.healthCheck)))
	app.Get("/status", adaptor.HTTPHandlerFunc(health.StatusHandler(s.healthCheck)))
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Webhook - AGORA 100% FIBER, SEM adaptor.HTTPHandlerFunc
	// Seu handler precisa ter assinatura Fiber: func(c *fiber.Ctx) error
	// Veja handlers_fiber.go de exemplo abaixo
	app.Get("/webhook", s.handler.HandleVerifyWebhookFiber)
	app.Post("/webhook", s.handler.HandleWebhookFiber)

	logger.GetLogger().Info("Webhook Fiber 100% - SEM sanitize.go, SEM Mux",
		zap.String("addr", addr),
	)
	return app.Listen(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
