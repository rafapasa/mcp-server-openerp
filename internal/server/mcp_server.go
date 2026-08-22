package server

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/helpers"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/server/tools"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

type MCPServer struct {
	mcp       *mcpserver.MCPServer
	sse       *mcpserver.SSEServer
	cardapio  service.CardapioServiceInterface
	pedido    service.PedidoServiceInterface
	carrinho  service.CarrinhoServiceInterface
	llmClient *llm.UnifiedLLM
	cfg       *config.Config
}

// NewMCPServer - Wire Ready - não cria repo, só recebe
func NewMCPServer(
	cfg *config.Config,
	cardapioService service.CardapioServiceInterface,
	pedidoService service.PedidoServiceInterface,
	carrinhoService service.CarrinhoServiceInterface,
	llmClient *llm.UnifiedLLM,
) *MCPServer {
	mcpSrv := mcpserver.NewMCPServer(
		"mcp-openerp",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(false, false),
		mcpserver.WithRecovery(),
	)

	s := &MCPServer{
		mcp:       mcpSrv,
		cardapio:  cardapioService,
		pedido:    pedidoService,
		carrinho:  carrinhoService,
		llmClient: llmClient,
		cfg:       cfg,
	}

	s.registerTools()

	// SSE opcional pra expor via HTTP
	s.sse = mcpserver.NewSSEServer(s.mcp,
		mcpserver.WithBaseURL("/mcp"),
	)

	return s
}

func (s *MCPServer) registerTools() {
	deps := &tools.Dependencies{
		CardapioService: s.cardapio,
		PedidoService:   s.pedido,
		CarrinhoService: s.carrinho,
		LLMClient:       s.llmClient,
	}
	tools.RegisterAllTools(s.mcp, deps)
}

// ========== MCP PROTOCOL ==========

func (s *MCPServer) ServeStdio() error {
	return mcpserver.ServeStdio(s.mcp)
}

func (s *MCPServer) SSEServer() *mcpserver.SSEServer {
	return s.sse
}

func (s *MCPServer) GetMCP() *mcpserver.MCPServer {
	return s.mcp
}

func (s *MCPServer) AddTool(tool mcp.Tool, handler mcpserver.ToolHandlerFunc) {
	s.mcp.AddTool(tool, handler)
}

// ========== MÉTODOS USADOS PELO WEBHOOK - AGORA DELEGAM PRO SERVICE ==========
// Mantidos por compatibilidade, mas o ideal é o webhook chamar service direto

func (s *MCPServer) GetCardapio(tenantID uint) ([]dto.ProdutoItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.cardapio.GetCardapio(ctx, tenantID)
}

func (s *MCPServer) ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*dto.IntencaoCliente, error) {
	return s.llmClient.ExtractIntent(ctx, mensagem, cardapio)
}

func (s *MCPServer) GetCarrinho(ctx context.Context, clienteID, tenantID uint) (*dto.Carrinho, error) {
	return s.carrinho.GetCarrinho(ctx, clienteID, tenantID)
}

func (s *MCPServer) AdicionarItemCarrinho(ctx context.Context, clienteID, tenantID uint, item dto.ItemCarrinho) error {
	return s.carrinho.AdicionarItem(ctx, clienteID, tenantID, item)
}

func (s *MCPServer) RemoverItemCarrinho(ctx context.Context, clienteID, tenantID uint, nome string, qtd int) error {
	return s.carrinho.RemoverItem(ctx, clienteID, tenantID, nome, qtd)
}

func (s *MCPServer) LimparCarrinho(ctx context.Context, clienteID, tenantID uint) error {
	return s.carrinho.LimparCarrinho(ctx, clienteID, tenantID)
}

func (s *MCPServer) FinalizarCarrinho(ctx context.Context, clienteID, tenantID uint, clienteNome string) (*dto.PedidoConfirmado, error) {
	return s.carrinho.FinalizarCarrinho(ctx, clienteID, tenantID, clienteNome)
}

// ========== FORMATAÇÃO - AGORA USA helpers/formatter.go (sem ciclo) ==========

func (s *MCPServer) FormatarResumoCarrinho(ctx context.Context, clienteID, tenantID uint) string {
	carrinho, err := s.carrinho.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao buscar carrinho: %v", err)
	}
	total := s.carrinho.CalcularTotal(carrinho)
	tempo := s.carrinho.CalcularTempoEstimado(carrinho)
	return helpers.FormatResumoCarrinho(carrinho.Itens, total, tempo)
}

func (s *MCPServer) FormatarRespostaPedido(ctx context.Context, pedido *dto.PedidoConfirmado) string {
	return helpers.FormatRespostaPedido(pedido)
}
