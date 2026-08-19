// internal/api/server.go
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

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

// Server representa o servidor API
type Server struct {
	httpServer  *http.Server
	handlers    *APIHandlers
	db          *gorm.DB
	cache       *redis.Client
	healthCheck *health.HealthChecker
}

// NewServer cria um novo servidor API
func NewServer(db *gorm.DB, cache *redis.Client) *Server {
	// Repositórios
	tenantRepo := repository.NewTenantRepository(db)
	produtoRepo := repository.NewProdutoRepository(db)
	pedidoRepo := repository.NewPedidoRepository(db)
	clienteRepo := repository.NewClienteRepository(db)
	enderecoRepo := repository.NewEnderecoRepository(db)

	// Serviços
	cardapioService := service.NewCardapioService(produtoRepo, tenantRepo, cache)
	pedidoService := service.NewPedidoService(pedidoRepo, cardapioService)
	clienteService := service.NewClienteService(clienteRepo, enderecoRepo)

	// Handlers
	handlers := NewAPIHandlers(clienteService, pedidoService, cardapioService)

	hc := health.NewDefaultHealthChecker(db, cache)

	return &Server{
		handlers:    handlers,
		db:          db,
		cache:       cache,
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

	if err := tracing.Init(tracingConfig); err != nil {
		logger.GetLogger().Warn("Erro ao iniciar tracing", zap.Error(err))
	}

	rateLimiter := middleware.NewRateLimiterFromEnv()

	// Cleanup periódico do rate limiter
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rateLimiter.Cleanup()
		}
	}()

	mux := http.NewServeMux()

	// Health checks
	mux.HandleFunc("GET /health", health.HealthHandler(s.healthCheck))
	mux.HandleFunc("GET /ready", health.ReadinessHandler(s.healthCheck))
	mux.HandleFunc("GET /status", health.StatusHandler(s.healthCheck))

	// Login (sem autenticação)
	mux.HandleFunc("POST /api/v1/login", s.handlers.Login)

	// API protegida
	mux.HandleFunc("GET /api/v1/dashboard", AuthMiddleware(s.handlers.Dashboard))
	mux.HandleFunc("GET /api/v1/pedidos", AuthMiddleware(s.handlers.ListPedidos))
	mux.HandleFunc("GET /api/v1/pedidos/{id}", AuthMiddleware(s.handlers.GetPedido))
	mux.HandleFunc("PATCH /api/v1/pedidos/{id}/status", AuthMiddleware(s.handlers.UpdatePedidoStatus))
	mux.HandleFunc("GET /api/v1/clientes", AuthMiddleware(s.handlers.ListClientes))
	mux.HandleFunc("GET /api/v1/clientes/{id}", AuthMiddleware(s.handlers.GetCliente))
	mux.HandleFunc("GET /api/v1/clientes/{id}/pedidos", AuthMiddleware(s.handlers.GetClientePedidos))
	mux.HandleFunc("GET /api/v1/clientes/{id}/enderecos", AuthMiddleware(s.handlers.GetClienteEnderecos))
	mux.HandleFunc("GET /api/v1/produtos", AuthMiddleware(s.handlers.ListProdutos))
	mux.HandleFunc("GET /api/v1/produtos/{id}", AuthMiddleware(s.handlers.GetProduto))

	// Métricas
	mux.Handle("GET /metrics", promhttp.Handler())

	// Middlewares
	handler := middleware.SecurityHeadersMiddleware(
		middleware.SanitizeMiddleware(
			middleware.RateLimitMiddleware(rateLimiter, middleware.TenantKeyExtractor)(
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

	logger.GetLogger().Info("API server started",
		zap.String("addr", addr),
		zap.Bool("tracing_enabled", cfg.TracingEnabled),
	)
	return s.httpServer.ListenAndServe()
}

// Shutdown finaliza o servidor gracefulmente
func (s *Server) Shutdown(ctx context.Context) error {
	logger.GetLogger().Info("Shutting down API server gracefully...")
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
