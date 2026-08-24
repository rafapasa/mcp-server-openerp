// internal/service/carrinho_service.go
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/cache"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/helpers"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"go.uber.org/zap"
)

const (
	TTLCarrinho = 3600 // 1 hora em segundos - escrita
)

type CarrinhoService struct {
	cache           *cache.Cache
	cardapioService CardapioServiceInterface
	pedidoService   PedidoServiceInterface
	produtoRepo     repository.ProdutoRepository
	llmClient       *llm.UnifiedLLM
	preprocessor    *llm.Preprocessor
}

func NewCarrinhoService(
	cache *cache.Cache,
	cardapioService CardapioServiceInterface,
	pedidoService PedidoServiceInterface,
	produtoRepo repository.ProdutoRepository,
	llmClient *llm.UnifiedLLM,
) CarrinhoServiceInterface {
	return &CarrinhoService{
		cache:           cache,
		cardapioService: cardapioService,
		pedidoService:   pedidoService,
		produtoRepo:     produtoRepo,
		llmClient:       llmClient,
		preprocessor:    llm.NewPreprocessor(),
	}
}

func (s *CarrinhoService) getKey(clienteID, tenantID string) string {
	return fmt.Sprintf("carrinho:%s:%s", tenantID, clienteID)
}

// GetCarrinho - FIX Issue #8: GetOrSet 2m, sem MySQL, <50ms no View
func (s *CarrinhoService) GetCarrinho(ctx context.Context, clienteID, tenantID uint) (*dto.Carrinho, error) {
	key := s.getKey(fmt.Sprint(clienteID), fmt.Sprint(tenantID))

	carrinho, err := cache.GetOrSet(s.cache, ctx, key, 2*time.Minute, func() (*dto.Carrinho, error) {
		// Fallback: cache miss -> carrinho vazio (carrinho vive só em Redis)
		c := &dto.Carrinho{
			ClienteID: fmt.Sprint(clienteID),
			TenantID:  fmt.Sprint(tenantID),
			Itens:     []dto.ItemCarrinho{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		logger.Debug(ctx, "Carrinho vazio recuperado - cache miss",
			zap.Uint("cliente_id", clienteID),
			zap.Uint("tenant_id", tenantID))
		return c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar carrinho: %w", err)
	}

	logger.Debug(ctx, "Carrinho recuperado",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("tenant_id", tenantID),
		zap.Int("itens_count", len(carrinho.Itens)))

	return carrinho, nil
}

func (s *CarrinhoService) saveCarrinho(carrinho *dto.Carrinho) error {
	carrinho.UpdatedAt = time.Now()
	key := s.getKey(carrinho.ClienteID, carrinho.TenantID)

	// Usa SetJSONWithContext do seu database.Redis (já faz Marshal interno)
	if err := s.cache.SetJSONWithContext(context.Background(), key, carrinho, TTLCarrinho*time.Second); err != nil {
		return fmt.Errorf("erro ao salvar carrinho: %w", err)
	}
	return nil
}

func (s *CarrinhoService) AdicionarItem(ctx context.Context, clienteID, tenantID uint, item dto.ItemCarrinho) error {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return err
	}
	for i, existingItem := range carrinho.Itens {
		if strings.EqualFold(existingItem.Nome, item.Nome) {
			carrinho.Itens[i].Quantidade += item.Quantidade
			if item.Observacao != "" {
				carrinho.Itens[i].Observacao = item.Observacao
			}
			logger.Info(ctx, "Item atualizado no carrinho",
				zap.Uint("cliente_id", clienteID),
				zap.Uint("tenant_id", tenantID),
				zap.String("item_nome", item.Nome),
				zap.Int("quantidade_nova", carrinho.Itens[i].Quantidade))
			return s.saveCarrinho(carrinho)
		}
	}
	carrinho.Itens = append(carrinho.Itens, item)
	logger.Info(ctx, "Item adicionado ao carrinho",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("tenant_id", tenantID),
		zap.String("item_nome", item.Nome),
		zap.Int("quantidade", item.Quantidade))
	return s.saveCarrinho(carrinho)
}

func (s *CarrinhoService) RemoverItem(ctx context.Context, clienteID, tenantID uint, nome string, quantidade int) error {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return err
	}
	for i, item := range carrinho.Itens {
		if strings.EqualFold(item.Nome, nome) {
			if quantidade == 0 || quantidade >= item.Quantidade {
				carrinho.Itens = append(carrinho.Itens[:i], carrinho.Itens[i+1:]...)
				logger.Info(ctx, "Item removido completamente do carrinho",
					zap.Uint("cliente_id", clienteID),
					zap.Uint("tenant_id", tenantID),
					zap.String("item_nome", nome))
			} else {
				carrinho.Itens[i].Quantidade -= quantidade
				logger.Info(ctx, "Quantidade de item reduzida no carrinho",
					zap.Uint("cliente_id", clienteID),
					zap.Uint("tenant_id", tenantID),
					zap.String("item_nome", nome),
					zap.Int("quantidade_restante", carrinho.Itens[i].Quantidade))
			}
			return s.saveCarrinho(carrinho)
		}
	}
	return fmt.Errorf("item '%s' não encontrado no carrinho", nome)
}

func (s *CarrinhoService) LimparCarrinho(ctx context.Context, clienteID, tenantID uint) error {
	key := s.getKey(fmt.Sprint(clienteID), fmt.Sprint(tenantID))
	logger.Info(ctx, "Carrinho limpo",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("tenant_id", tenantID))
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
	tempoBase := 15
	tempoPorItem := 5
	totalItems := 0
	for _, item := range carrinho.Itens {
		totalItems += item.Quantidade
	}
	return tempoBase + (totalItems * tempoPorItem)
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
			Nome:          item.Nome,
			Quantidade:    item.Quantidade,
			Observacao:    item.Observacao,
			PrecoUnitario: item.Preco,
		})
	}
	pedidoConfirmado, err := s.pedidoService.ProcessarPedido(ctx, tenantID, clienteID, clienteNome, pedidoExtraido)
	if err != nil {
		return nil, err
	}
	if err := s.LimparCarrinho(ctx, clienteID, tenantID); err != nil {
		logger.Error(ctx, "Erro ao limpar carrinho após finalizar",
			zap.Error(err),
			zap.Uint("cliente_id", clienteID),
			zap.Uint("tenant_id", tenantID),
			zap.Int("pedido_id", pedidoConfirmado.ID))
	}
	return pedidoConfirmado, nil
}

func (s *CarrinhoService) ProcessarMensagem(ctx context.Context, clienteID, tenantID uint, mensagem string) (*dto.Carrinho, error) {
	logger.Info(
		ctx, "[Carrinho] Processando mensagem",
		zap.Uint("Client_id", clienteID),
		zap.Uint("Tenant_id", tenantID),
		zap.String("Mensagem", mensagem),
	)
	pp := s.preprocessor.Process(mensagem)
	logger.Debug(
		ctx, "Mensagem pré-processada para LLM",
		zap.String("original", pp.Original),
		zap.String("cleaned", pp.Cleaned),
		zap.Strings("medidas", pp.Medidas),
	)
	intencao, err := s.llmClient.ExtractIntent(ctx, pp.Cleaned, []dto.ProdutoItem{})
	if err != nil {
		logger.Error(
			ctx, "Erro ao extrair intenção",
			zap.Error(err),
			zap.String("mensagem", mensagem),
		)
		return nil, fmt.Errorf("erro ao extrair intenção: %w", err)
	}
	logger.Info(
		ctx, "Intenção detectada pelo LLM",
		zap.String("acao", intencao.Acao),
		zap.Int("itens_count", len(intencao.Itens)),
	)
	if intencao.Acao != "adicionar" && intencao.Acao != "add" {
		return s.GetCarrinho(ctx, clienteID, tenantID)
	}
	nomesProdutos := make([]string, len(intencao.Itens))
	for i, item := range intencao.Itens {
		nomesProdutos[i] = item.Nome
	}
	logger.Debug(ctx, "Buscando produtos no banco de dados",
		zap.Uint("tenant_id", tenantID),
		zap.Strings("nomes_produtos", nomesProdutos),
		zap.Int("quantidade_nomes", len(nomesProdutos)))
	produtosEncontrados, err := s.produtoRepo.BuscarProdutosLote(ctx, fmt.Sprint(tenantID), nomesProdutos)
	if err != nil {
		logger.Error(
			ctx, "Erro ao buscar produtos",
			zap.Error(err),
			zap.Uint("tenant_id: ", tenantID),
			zap.Strings("produtos", nomesProdutos),
		)
		return nil, fmt.Errorf("erro ao buscar produtos: %w", err)
	}
	var encontrados []dto.ItemCarrinho
	var naoEncontrados []string
	for _, item := range intencao.Itens {
		if produto, ok := produtosEncontrados[item.Nome]; ok {
			encontrados = append(encontrados, dto.ItemCarrinho{
				Nome:       produto.Nome,
				Quantidade: item.Quantidade,
				Observacao: item.Observacao,
				Preco:      produto.Preco,
			})
			logger.Info(ctx, "Produto encontrado no cardápio", zap.String("item_original", item.Nome), zap.String("produto_encontrado", produto.Nome), zap.Float64("preco", produto.Preco))
		} else {
			naoEncontrados = append(naoEncontrados, item.Nome)
			logger.Warn(ctx, "Produto não encontrado no cardápio", zap.String("item_nome", item.Nome))
		}
	}
	if len(naoEncontrados) > 0 && len(produtosEncontrados) > 0 {
		logger.Warn(ctx, "Produtos não encontrados, tentando corrigir",
			zap.Strings("nomes_nao_encontrados", naoEncontrados))
		similares, err := s.produtoRepo.BuscarProdutosLote(ctx, fmt.Sprint(tenantID), naoEncontrados)
		if err != nil {
			logger.Error(ctx, "Erro ao buscar produtos similares para correção", zap.Error(err), zap.Uint("tenant_id", tenantID), zap.Strings("nomes_nao_encontrados", naoEncontrados))
		}
		if len(similares) > 0 {
			corrigidos, err := llm.CorrigirNomes(ctx, naoEncontrados, similares, s.llmClient.GenerateResponse)
			if err != nil {
				logger.Error(ctx, "Erro ao corrigir nomes de produtos com LLM", zap.Error(err), zap.Strings("nomes_nao_encontrados", naoEncontrados))
			} else {
				for _, item := range corrigidos {
					if produto, ok := similares[item.Nome]; ok {
						encontrados = append(encontrados, dto.ItemCarrinho{
							Nome:       produto.Nome,
							Observacao: item.Observacao,
							Preco:      produto.Preco,
						})
						logger.Info(ctx, "Produto corrigido e adicionado", zap.String("nome_original", item.Nome), zap.String("nome_corrigido", produto.Nome))
					}
				}
			}
		}
	}
	_, err = s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return nil, err
	}
	for _, item := range encontrados {
		if err := s.AdicionarItem(ctx, clienteID, tenantID, item); err != nil {
			logger.Error(ctx, "Erro ao adicionar item corrigido ao carrinho",
				zap.Error(err),
				zap.String("item_nome", item.Nome),
				zap.Uint("cliente_id", clienteID),
				zap.Uint("tenant_id", tenantID))
		}
	}
	return s.GetCarrinho(ctx, clienteID, tenantID)
}

func (s *CarrinhoService) BuscarProdutos(ctx context.Context, tenantID, termo string, limit int) ([]dto.ProdutoItem, error) {
	return s.produtoRepo.BuscarProdutosPorNome(ctx, tenantID, termo, limit)
}

func (s *CarrinhoService) BuscarProdutosLote(ctx context.Context, tenantID string, nomes []string) (map[string]dto.ProdutoItem, error) {
	return s.produtoRepo.BuscarProdutosLote(ctx, tenantID, nomes)
}

func (s *CarrinhoService) FormatResumoCarrinho(ctx context.Context, clienteID, tenantID uint) (string, error) {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return "", err
	}
	total := s.CalcularTotal(carrinho)
	tempo := s.CalcularTempoEstimado(carrinho)
	return helpers.FormatResumoCarrinho(carrinho.Itens, total, tempo), nil
}

func (s *CarrinhoService) FormatarPedidoConfirmado(pedido *dto.PedidoConfirmado) string {
	return helpers.FormatRespostaPedido(pedido)
}
