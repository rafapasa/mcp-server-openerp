// internal/webhook/routes.go
package webhook

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/health"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/metrics"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/middleware"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/tracing"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Server servidor webhook
type Server struct {
	httpServer *http.Server
	handler    *WebhookHandler
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

	return &Server{
		handler: handler,
	}
}

// Start inicia o servidor HTTP
func (s *Server) Start(addr string, db *gorm.DB, cache *redis.Client, llmClient llm.LLMClient) error {
	metrics.Init()

	// Inicializa tracing
	if err := tracing.Init(tracing.DefaultConfig()); err != nil {
		println("Erro ao iniciar tracing: " + err.Error())
	}

	// Cria health checker com verificações padrão
	hc := health.NewDefaultHealthChecker(db, cache, llmClient)

	mux := http.NewServeMux()

	// Health checks
	mux.HandleFunc("GET /health", health.HealthHandler(hc))
	mux.HandleFunc("GET /ready", health.ReadinessHandler(hc))
	mux.HandleFunc("GET /status", health.StatusHandler(hc))

	// Rotas do webhook
	mux.HandleFunc("GET /webhook", s.handler.HandleVerifyWebhook)
	mux.HandleFunc("POST /webhook", s.handler.HandleWebhook)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.Handle("GET /metrics", promhttp.Handler())

	// Aplica middlewares (logging + métricas + tracing)
	handler := middleware.TracingMiddleware(
		middleware.LoggingMiddleware(
			middleware.MetricsMiddleware(mux),
		),
	)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	log.Printf("✅ Webhook rodando em %s", addr)
	log.Printf("   GET  /webhook - Verificação do WhatsApp")
	log.Printf("   POST /webhook - Recebimento de mensagens")
	log.Printf("   GET  /health  - Health check")

	return s.httpServer.ListenAndServe()
}
