// internal/service/carrinho_service.go - CLEAN
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/cache"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/helpers"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"go.uber.org/zap"
)

const TTLCarrinho = 3600

type CarrinhoService struct {
	cache           *cache.Cache
	cardapioService CardapioServiceInterface
	pedidoService   PedidoServiceInterface
	produtoRepo     repository.ProdutoRepository
	llmService      LLMServiceInterface
}

func NewCarrinhoService(
	cache *cache.Cache,
	cardapioService CardapioServiceInterface,
	pedidoService PedidoServiceInterface,
	produtoRepo repository.ProdutoRepository,
	llmService LLMServiceInterface,
) CarrinhoServiceInterface {
	return &CarrinhoService{
		cache:           cache,
		cardapioService: cardapioService,
		pedidoService:   pedidoService,
		produtoRepo:     produtoRepo,
		llmService:      llmService,
	}
}

func (s *CarrinhoService) getKey(clienteID, tenantID uint) string {
	return fmt.Sprintf("carrinho:%d:%d", tenantID, clienteID)
}

func (s *CarrinhoService) GetCarrinho(ctx context.Context, clienteID, tenantID uint) (*dto.Carrinho, error) {
	key := s.getKey(clienteID, tenantID)
	carrinho, err := cache.GetOrSet(s.cache, ctx, key, 2*time.Minute, func() (*dto.Carrinho, error) {
		return &dto.Carrinho{
			ClienteID: fmt.Sprint(clienteID),
			TenantID:  fmt.Sprint(tenantID),
			Itens:     []dto.ItemCarrinho{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar carrinho: %w", err)
	}
	return carrinho, nil
}

func (s *CarrinhoService) saveCarrinho(ctx context.Context, carrinho *dto.Carrinho) error {
	carrinho.UpdatedAt = time.Now()
	key := s.getKey(parseUint(carrinho.ClienteID), parseUint(carrinho.TenantID))
	if err := s.cache.SetJSONWithContext(ctx, key, carrinho, TTLCarrinho*time.Second); err != nil {
		return fmt.Errorf("erro ao salvar carrinho: %w", err)
	}
	return nil
}

// ProcessarMensagem - DONO DO WORKFLOW (cardápio pequeno via ResolveItemsByMenu)
func (s *CarrinhoService) ProcessarMensagem(ctx context.Context, clienteID, tenantID uint, input dto.MessageInput) (string, error) {
	logger.Info(
		ctx, "[CARRINHO] Processando mensagem",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("tenant_id", tenantID),
		zap.String("source", string(input.Source)),
	)

	cardapio, err := s.cardapioService.GetCardapio(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, err.Error())
		return "", fmt.Errorf("erro cardápio: %w", err)
	}

	intencao, err := llm.RetryWithBackoff(ctx, llm.DefaultRetryConfig(), func() (*dto.IntencaoCliente, error) {
		return s.llmService.ResolveItemsByMenu(ctx, tenantID, input, cardapio)
	})
	if err != nil {
		logger.Error(ctx, "Erro ResolveItemsByMenu", zap.Error(err))
		return "⚠️ Tive um problema técnico, tenta de novo por favor.", nil
	}

	logger.Info(
		ctx, "Intenção detectada",
		zap.String("acao", intencao.Acao),
		zap.Int("itens", len(intencao.Itens)),
	)

	switch intencao.Acao {
	case "conversa":
		if intencao.Resposta != "" {
			return intencao.Resposta, nil
		}
		return "Olá! 😊 Como posso ajudar?", nil

	case "visualizar", "ver", "mostrar", "carrinho":
		return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)

	case "limpar", "clear", "esvaziar":
		_ = s.LimparCarrinho(ctx, clienteID, tenantID)
		return "🗑️ Carrinho limpo!", nil

	case "listar_categorias":
		return s.cardapioService.ListarCategoriasHumanizado(cardapio), nil

	case "listar_produtos":
		return s.cardapioService.ListarProdutosHumanizado(cardapio, intencao.Filtro), nil

	case "remover", "remove", "tirar":
		if len(intencao.Itens) == 0 {
			return "O que você quer remover?", nil
		}
		for _, it := range intencao.Itens {
			_ = s.RemoverItem(ctx, clienteID, tenantID, it, it.Quantidade)
		}
		return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)

	case "finalizar", "confirmar", "fechar", "checkout":
		pedido, err := s.FinalizarCarrinho(ctx, clienteID, tenantID, "")
		if err != nil {
			if err.Error() == "carrinho vazio" {
				return "🛒 Seu carrinho está vazio. Me diga o que quer adicionar!", nil
			}
			return "❌ Erro ao finalizar pedido.", nil
		}
		return s.FormatarPedidoConfirmado(pedido), nil

	default: // adicionar
		if len(intencao.Itens) == 0 {
			return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)
		}
		carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			return "", err
		}
		for _, item := range intencao.Itens {
			carrinho = s.mergeItem(carrinho, item)
		}
		if err := s.saveCarrinho(ctx, carrinho); err != nil {
			return "", err
		}
		return s.FormatResumoCarrinho(ctx, carrinho)
	}
}

func (s *CarrinhoService) mergeItem(carrinho *dto.Carrinho, item dto.ItemCarrinho) *dto.Carrinho {
	for i, existing := range carrinho.Itens {
		if existing.ProdutoItem.ID == item.ProdutoItem.ID {
			carrinho.Itens[i].Quantidade += item.Quantidade
			if item.Observacao != "" {
				carrinho.Itens[i].Observacao = item.Observacao
			}
			return carrinho
		}
	}
	carrinho.Itens = append(carrinho.Itens, item)
	return carrinho
}

func (s *CarrinhoService) AdicionarItem(ctx context.Context, clienteID, tenantID uint, item dto.ItemCarrinho) error {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return err
	}
	carrinho = s.mergeItem(carrinho, item)
	return s.saveCarrinho(ctx, carrinho)
}

func (s *CarrinhoService) RemoverItem(ctx context.Context, clienteID, tenantID uint, itemCarrinho dto.ItemCarrinho, quantidade int) error {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return err
	}
	for i, item := range carrinho.Itens {
		if item.ProdutoItem.ID == itemCarrinho.ProdutoItem.ID {
			if quantidade == 0 || quantidade >= item.Quantidade {
				carrinho.Itens = append(carrinho.Itens[:i], carrinho.Itens[i+1:]...)
			} else {
				carrinho.Itens[i].Quantidade -= quantidade
			}
			return s.saveCarrinho(ctx, carrinho)
		}
	}
	return fmt.Errorf("item '%s' não encontrado", itemCarrinho.ProdutoItem.Nome)
}

func (s *CarrinhoService) LimparCarrinho(ctx context.Context, clienteID, tenantID uint) error {
	key := s.getKey(clienteID, tenantID)
	return s.cache.DeleteWithContext(ctx, key)
}

func (s *CarrinhoService) CalcularTotal(carrinho *dto.Carrinho) float64 {
	total := 0.0
	for _, item := range carrinho.Itens {
		total += item.Preco * float64(item.Quantidade)
	}
	return total
}

func (s *CarrinhoService) CalcularTempoEstimado(carrinho *dto.Carrinho) int {
	if len(carrinho.Itens) == 0 {
		return 0
	}
	total := 0
	for _, item := range carrinho.Itens {
		total += item.Quantidade
	}
	return 15 + (total * 5)
}

func (s *CarrinhoService) FinalizarCarrinho(ctx context.Context, clienteID, tenantID uint, clienteNome string) (*dto.PedidoConfirmado, error) {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return nil, err
	}
	if len(carrinho.Itens) == 0 {
		return nil, fmt.Errorf("carrinho vazio")
	}
	pedidoExtraido := &dto.PedidoExtraido{}
	for _, item := range carrinho.Itens {
		pedidoExtraido.Itens = append(pedidoExtraido.Itens, dto.ItemPedidoInput{
			ProdutoItem:   item.ProdutoItem,
			Quantidade:    item.Quantidade,
			Observacao:    item.Observacao,
			PrecoUnitario: item.Preco,
		})
	}
	confirmado, err := s.pedidoService.ProcessarPedido(ctx, tenantID, clienteID, clienteNome, pedidoExtraido)
	if err != nil {
		return nil, err
	}
	_ = s.LimparCarrinho(ctx, clienteID, tenantID)
	return confirmado, nil
}

func (s *CarrinhoService) BuscarProdutos(ctx context.Context, tenantID, termo string, limit int) ([]dto.ProdutoItem, error) {
	return s.produtoRepo.BuscarProdutosPorNome(ctx, tenantID, termo, limit)
}

func (s *CarrinhoService) BuscarProdutosLote(ctx context.Context, tenantID string, nomes []string) (map[string]dto.ProdutoItem, error) {
	return s.produtoRepo.BuscarProdutosLote(ctx, tenantID, nomes)
}

func (s *CarrinhoService) FormatResumoCarrinhoByCliente(ctx context.Context, clienteID, tenantID uint) (string, error) {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return "", err
	}
	return s.FormatResumoCarrinho(ctx, carrinho)
}

func (s *CarrinhoService) FormatResumoCarrinho(ctx context.Context, carrinho *dto.Carrinho) (string, error) {
	total := s.CalcularTotal(carrinho)
	tempo := s.CalcularTempoEstimado(carrinho)
	return helpers.FormatResumoCarrinho(carrinho.Itens, total, tempo), nil
}

func (s *CarrinhoService) FormatarPedidoConfirmado(pedido *dto.PedidoConfirmado) string {
	return helpers.FormatRespostaPedido(pedido)
}

func parseUint(s string) uint {
	var v uint
	fmt.Sscan(s, &v)
	return v
}
