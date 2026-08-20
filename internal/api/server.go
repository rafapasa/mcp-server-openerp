// internal/api/server.go - FINAL DEFINITIVO 100% FIBER - SEM ADAPTOR, SEM MUX
package api

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rafapasa/mcp-server-openerp/internal/cache"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/health"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/metrics"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/middleware"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/tracing"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Server struct {
	app         *fiber.App
	handlers    *APIHandlers
	db          *gorm.DB
	cache       *redis.Client
	healthCheck *health.HealthChecker
}

func NewServer(db *gorm.DB, cacheClient *redis.Client) *Server {
	tenantRepo := repository.NewTenantRepository(db)
	produtoRepo := repository.NewProdutoRepository(db)
	pedidoRepo := repository.NewPedidoRepository(db)
	clienteRepo := repository.NewClienteRepository(db)
	enderecoRepo := repository.NewEnderecoRepository(db)

	cardapioService := service.NewCardapioService(produtoRepo, tenantRepo, cacheClient)
	pedidoService := service.NewPedidoService(pedidoRepo, cardapioService)
	clienteService := service.NewClienteService(clienteRepo, enderecoRepo)

	handlers := NewAPIHandlers(clienteService, pedidoService, cardapioService)
	handlers.SetCache(cache.New(cacheClient))

	hc := health.NewDefaultHealthChecker(db, cacheClient)

	return &Server{
		handlers:    handlers,
		db:          db,
		cache:       cacheClient,
		healthCheck: hc,
	}
}

func (s *Server) Start(addr string) error {
	metrics.Init()
	cfg := config.LoadConfigOrDefault()
	tracing.Init(tracing.Config{
		Enabled: cfg.TracingEnabled, Endpoint: cfg.TracingEndpoint,
		ServiceName: cfg.TracingServiceName, SamplingRate: cfg.TracingSamplingRate,
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
		AppName: "MCP API Fiber 100%", ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Tenant-ID",
	}))

	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		return c.Next()
	})

	app.Use(func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/health") {
			return c.Next()
		}
		if ok, _ := rateLimiter.Allow(c.Get("X-Tenant-ID", c.IP())); !ok {
			return c.Status(429).JSON(fiber.Map{"error": "rate limit"})
		}
		return c.Next()
	})

	s.app = app

	// Health Fiber puro
	app.Get("/health", func(c *fiber.Ctx) error {
		// chama seu healthcheck direto, sem adaptor
		if err := s.db.Exec("SELECT 1").Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "down"})
		}
		return c.JSON(fiber.Map{"status": "up"})
	})
	app.Get("/ready", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ready"})
	})

	v1 := app.Group("/api/v1")
	v1.Post("/login", s.handlers.LoginFiber)

	// Protegidas com Auth Fiber
	protected := v1.Group("", AuthMiddlewareFiber())
	protected.Get("/dashboard", s.handlers.DashboardFiber)
	protected.Get("/pedidos", s.handlers.ListPedidosFiber)
	protected.Get("/pedidos/:id", s.handlers.GetPedidoFiber)
	protected.Patch("/pedidos/:id/status", s.handlers.UpdatePedidoStatusFiber)
	protected.Get("/clientes", s.handlers.ListClientesFiber)
	protected.Get("/clientes/:id", s.handlers.GetClienteFiber)
	protected.Get("/clientes/:id/pedidos", s.handlers.GetClientePedidosFiber)
	protected.Get("/clientes/:id/enderecos", s.handlers.GetClienteEnderecosFiber)
	protected.Get("/produtos", s.handlers.ListProdutosFiber)
	protected.Get("/produtos/:id", s.handlers.GetProdutoFiber)

	logger.GetLogger().Info("API Fiber 100% sem Mux sem adaptor", zap.String("addr", addr))
	return app.Listen(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
