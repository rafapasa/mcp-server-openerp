// internal/webhook/routes.go
package webhook

import (
	"log"
	"net/http"

	"github.com/rafapasa/mcp-server-openerp/internal/llm"
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
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// Rotas do webhook
	mux.HandleFunc("GET /webhook", s.handler.HandleVerifyWebhook)
	mux.HandleFunc("POST /webhook", s.handler.HandleWebhook)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("✅ Webhook rodando em %s", addr)
	log.Printf("   GET  /webhook - Verificação do WhatsApp")
	log.Printf("   POST /webhook - Recebimento de mensagens")
	log.Printf("   GET  /health  - Health check")

	return s.httpServer.ListenAndServe()
}
