// internal/webhook/routes.go
package webhook

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
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

// Server servidor webhook
type Server struct {
	httpServer  *http.Server
	handler     *WebhookHandler
	db          *gorm.DB
	cache       *redis.Client
	llmClient   llm.LLMClient
	healthCheck *health.HealthChecker
}

// NewServer cria um novo servidor webhook
func NewServer(db *gorm.DB, cache *redis.Client, llmClient llm.LLMClient) *Server {
	// Inicializa repositórios
	tenantRepo := repository.NewTenantRepository(db)
	produtoRepo := repository.NewProdutoRepository(db)
	pedidoRepo := repository.NewPedidoRepository(db)

	// Inicializa serviços
	cardapioService := service.NewCardapioService(produtoRepo, tenantRepo, cache)
	pedidoService := service.NewPedidoService(pedidoRepo, cardapioService)
	_ = service.NewCarrinhoService(cache, cardapioService, pedidoService, produtoRepo, llmClient)

	// Cria MCP Server (para reutilizar a lógica)
	mcpServer := server.NewMCPServer(db, cache, llmClient)

	// Cria cliente WhatsApp
	whatsApp := NewWhatsAppClient()

	// Cria handler
	handler := NewWebhookHandler(mcpServer, whatsApp)

	// Cria health checker
	hc := health.NewDefaultHealthChecker(db, cache, llmClient)

	return &Server{
		handler:     handler,
		db:          db,
		cache:       cache,
		llmClient:   llmClient,
		healthCheck: hc,
	}
}

// Start inicia o servidor HTTP
func (s *Server) Start(addr string) error {

	// Inicializa métricas
	metrics.Init()

	// Carrega configuração de tracing do .env
	cfg := config.LoadConfigOrDefault()
	tracingConfig := tracing.Config{
		Enabled:      cfg.TracingEnabled,
		Endpoint:     cfg.TracingEndpoint,
		ServiceName:  cfg.TracingServiceName,
		SamplingRate: cfg.TracingSamplingRate,
	}

	// Inicializa tracing
	if err := tracing.Init(tracingConfig); err != nil {
		logger.GetLogger().Warn("Erro ao iniciar tracing", zap.Error(err))
	}

	mux := http.NewServeMux()

	// Health checks (usando o healthChecker com contexto)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		health.HealthHandler(s.healthCheck)(w, r)
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		health.ReadinessHandler(s.healthCheck)(w, r)
	})
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		health.StatusHandler(s.healthCheck)(w, r)
	})

	// Rotas do webhook
	mux.HandleFunc("GET /webhook", s.handler.HandleVerifyWebhook)
	mux.HandleFunc("POST /webhook", s.handler.HandleWebhook)

	// Métricas
	mux.Handle("GET /metrics", promhttp.Handler())

	// Aplica middlewares (logging + métricas + tracing)
	handler := middleware.TracingMiddleware(
		middleware.LoggingMiddleware(
			middleware.MetricsMiddleware(mux),
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
		zap.Bool("tracing_enabled", tracingConfig.Enabled),
	)
	logger.GetLogger().Info("Rotas disponíveis",
		zap.String("GET /webhook", "Verificação do WhatsApp"),
		zap.String("POST /webhook", "Recebimento de mensagens"),
		zap.String("GET /health", "Health check"),
		zap.String("GET /ready", "Readiness check"),
		zap.String("GET /status", "Status detalhado"),
		zap.String("GET /metrics", "Métricas Prometheus"),
	)

	return s.httpServer.ListenAndServe()
}

// Shutdown finaliza o servidor gracefully
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
