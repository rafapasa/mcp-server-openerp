package server

import (
	"context"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/health"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/metrics"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/middleware"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/tracing"
	serverMidleware "github.com/rafapasa/mcp-server-openerp/internal/server/middleware"
	"github.com/rafapasa/mcp-server-openerp/internal/webhook"
	"go.uber.org/zap"
)

// HttpServer é o Fiber unificado - API + Webhook + Health + MCP SSE
type HttpServer struct {
	app            *fiber.App
	mcpServer      *MCPServer
	apiHandlers    *APIHandlers
	webhookHandler *webhook.WebhookHandler
	healthCheck    *health.HealthChecker
	cfg            *config.Config
}

// NewHttpServer - 100% Wire Ready - NÃO CRIA REPO/SERVICE DENTRO
func NewHttpServer(
	cfg *config.Config,
	mcpServer *MCPServer,
	apiHandlers *APIHandlers,
	webhookHandler *webhook.WebhookHandler,
	healthCheck *health.HealthChecker,
) *HttpServer {
	// Wire não chama SetCache automaticamente, injeta manual aqui

	return &HttpServer{
		mcpServer:      mcpServer,
		apiHandlers:    apiHandlers,
		webhookHandler: webhookHandler,
		healthCheck:    healthCheck,
		cfg:            cfg,
	}
}

func (s *HttpServer) buildFiber() *fiber.App {
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
		AppName:      "MCP Universal - 1 Binary",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Tenant-ID",
	}))
	app.Use(middleware.SecurityHeadersFiber(s.cfg))
	app.Use(middleware.TracingFiber())
	app.Use(middleware.MetricsFiber())
	app.Use(middleware.LoggerFiber())
	app.Use(middleware.RateLimitFiber(rateLimiter))

	// 1. HEALTH & METRICS
	app.Get("/health", adaptor.HTTPHandlerFunc(health.HealthHandler(s.healthCheck)))
	app.Get("/ready", adaptor.HTTPHandlerFunc(health.ReadinessHandler(s.healthCheck)))
	app.Get("/status", adaptor.HTTPHandlerFunc(health.StatusHandler(s.healthCheck)))
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// 2. WEBHOOK WHATSAPP - 100% Fiber
	app.Get("/webhook", s.webhookHandler.HandleWebhookFiber)
	app.Post("/webhook", s.webhookHandler.HandleWebhookFiber)

	// 3. API PAINEL
	v1 := app.Group("/api/v1")
	v1.Post("/login", s.apiHandlers.LoginFiber)

	protected := v1.Group("", serverMidleware.AuthMiddlewareFiber(s.apiHandlers.authService))
	protected.Get("/dashboard", s.apiHandlers.DashboardFiber)
	protected.Get("/pedidos", s.apiHandlers.ListPedidosFiber)
	protected.Get("/pedidos/:id", s.apiHandlers.GetPedidoFiber)
	protected.Patch("/pedidos/:id/status", s.apiHandlers.UpdatePedidoStatusFiber)
	protected.Get("/clientes", s.apiHandlers.ListClientesFiber)
	protected.Get("/clientes/:id", s.apiHandlers.GetClienteFiber)
	protected.Get("/clientes/:id/pedidos", s.apiHandlers.GetClientePedidosFiber)
	protected.Get("/clientes/:id/enderecos", s.apiHandlers.GetClienteEnderecosFiber)
	protected.Get("/produtos", s.apiHandlers.ListProdutosFiber)
	protected.Get("/produtos/:id", s.apiHandlers.GetProdutoFiber)
	protected.Get("/formas-pagamento", s.apiHandlers.ListFormasPagamentoFiber)
	protected.Get("/formas-pagamento/:id", s.apiHandlers.GetFormaPagamentoFiber)
	protected.Post("/formas-pagamento", s.apiHandlers.CreateFormaPagamentoFiber)
	protected.Put("/formas-pagamento/:id", s.apiHandlers.UpdateFormaPagamentoFiber)
	protected.Delete("/formas-pagamento/:id", s.apiHandlers.DeleteFormaPagamentoFiber)

	// 4. MCP SSE (se você expõe MCP via HTTP)
	// Se seu MCPServer tem SSEHandler, descomenta:
	// app.Get("/mcp/sse", adaptor.HTTPHandler(s.mcpServer.SSEHandler()))
	// app.Post("/mcp/messages", adaptor.HTTPHandler(s.mcpServer.MessageHandler()))

	logger.GetLogger().Info(
		"HttpServer Fiber montado",
		zap.Strings("routes", []string{"/health", "/ready", "/metrics", "/webhook", "/api/v1", "/mcp"}),
	)

	return app
}

func (s *HttpServer) Start(addr string) error {
	s.app = s.buildFiber()
	return s.app.Listen(addr)
}

func (s *HttpServer) Shutdown(ctx context.Context) error {
	if s.app == nil {
		return nil
	}
	return s.app.ShutdownWithContext(ctx)
}
