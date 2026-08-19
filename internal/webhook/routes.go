// internal/webhook/routes.go
package webhook

import (
	"context"
	"net/http"
	"time"

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
	httpServer  *http.Server
	handler     *WebhookHandler
	db          *gorm.DB
	cache       *redis.Client
	llmClient   llm.LLMClient
	transcriber *media.GroqTranscriber
	geminiLLM   llm.LLMClient
	deepseekLLM llm.LLMClient
	healthCheck *health.HealthChecker
}

func NewServer(db *gorm.DB, cache *redis.Client, llmClient llm.LLMClient, transcriber *media.GroqTranscriber, geminiLLM llm.LLMClient, deepseekLLM llm.LLMClient) *Server {
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

	// agora handler com 3 LLMs
	handler := NewWebhookHandler(mcpServer, whatsApp, clienteService, transcriber, geminiLLM, deepseekLLM)

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
	}
}

func (s *Server) Start(addr string) error {
	metrics.Init()

	cfg := config.LoadConfigOrDefault()
	tracingConfig := tracing.Config{
		Enabled:      cfg.TracingEnabled,
		Endpoint:     cfg.TracingEndpoint,
		ServiceName:  cfg.TracingServiceName,
		SamplingRate: cfg.TracingSamplingRate,
	}

	if err := tracing.Init(tracingConfig); err != nil {
		logger.GetLogger().Warn("Erro ao iniciar tracing", zap.Error(err))
	}

	rateLimiter := middleware.NewRateLimiterFromEnv()
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rateLimiter.Cleanup()
		}
	}()

	tenantExtractor := middleware.TenantKeyExtractor

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		health.HealthHandler(s.healthCheck)(w, r)
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		health.ReadinessHandler(s.healthCheck)(w, r)
	})
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		health.StatusHandler(s.healthCheck)(w, r)
	})

	mux.HandleFunc("GET /webhook", s.handler.HandleVerifyWebhook)
	mux.HandleFunc("POST /webhook", VerifyWebhookRequest(s.handler.HandleWebhook))

	mux.Handle("GET /metrics", promhttp.Handler())

	handler := middleware.SecurityHeadersMiddleware(
		middleware.SanitizeMiddleware(
			middleware.RateLimitMiddleware(rateLimiter, tenantExtractor)(
				middleware.APIHeadersMiddleware(
					middleware.TracingMiddleware(
						middleware.LoggingMiddleware(
							middleware.MetricsMiddleware(mux),
						),
					),
				),
			),
		),
	)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.GetLogger().Info("Webhook server configurado",
		zap.String("addr", addr),
		zap.Bool("hsts_enabled", middleware.IsProduction()),
	)

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger.GetLogger().Info("Webhook server desligando...")
	return s.httpServer.Shutdown(ctx)
}
