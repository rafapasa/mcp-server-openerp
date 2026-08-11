// internal/service/carrinho_service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/redis/go-redis/v9"
)

const (
	TTLCarrinho = 3600 // 1 hora em segundos
)

// CarrinhoService gerencia operações do carrinho no Redis
type CarrinhoService struct {
	cache           *redis.Client
	cardapioService CardapioServiceInterface
	pedidoService   PedidoServiceInterface
	produtoRepo     repository.ProdutoRepository
	llmClient       llm.LLMClient
	preprocessor    *llm.Preprocessor
}

// NewCarrinhoService cria um novo service de carrinho
func NewCarrinhoService(
	cache *redis.Client,
	cardapioService CardapioServiceInterface,
	pedidoService PedidoServiceInterface,
	produtoRepo repository.ProdutoRepository,
	llmClient llm.LLMClient,
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

// getKey retorna a chave do Redis para o carrinho
func (s *CarrinhoService) getKey(clienteID, tenantID string) string {
	return fmt.Sprintf("carrinho:%s:%s", tenantID, clienteID)
}

// GetCarrinho busca o carrinho do cliente
func (s *CarrinhoService) GetCarrinho(ctx context.Context, clienteID, tenantID string) (*dto.Carrinho, error) {
	key := s.getKey(clienteID, tenantID)

	data, err := s.cache.Get(ctx, key).Result()
	if err == redis.Nil {
		// Carrinho vazio
		return &dto.Carrinho{
			ClienteID: clienteID,
			TenantID:  tenantID,
			Itens:     []dto.ItemCarrinho{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar carrinho: %w", err)
	}

	var carrinho dto.Carrinho
	if err := json.Unmarshal([]byte(data), &carrinho); err != nil {
		return nil, fmt.Errorf("erro ao desserializar carrinho: %w", err)
	}

	return &carrinho, nil
}

// saveCarrinho salva o carrinho no Redis
func (s *CarrinhoService) saveCarrinho(carrinho *dto.Carrinho) error {
	carrinho.UpdatedAt = time.Now()

	data, err := json.Marshal(carrinho)
	if err != nil {
		return fmt.Errorf("erro ao serializar carrinho: %w", err)
	}

	key := s.getKey(carrinho.ClienteID, carrinho.TenantID)

	if err := s.cache.Set(context.Background(), key, data, TTLCarrinho*time.Second).Err(); err != nil {
		return fmt.Errorf("erro ao salvar carrinho: %w", err)
	}

	return nil
}

// AdicionarItem adiciona um item ao carrinho
func (s *CarrinhoService) AdicionarItem(ctx context.Context, clienteID, tenantID string, item dto.ItemCarrinho) error {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return err
	}

	// Verifica se o item já existe no carrinho
	for i, existingItem := range carrinho.Itens {
		if strings.EqualFold(existingItem.Nome, item.Nome) {
			carrinho.Itens[i].Quantidade += item.Quantidade
			// Mantém a observação mais recente
			if item.Observacao != "" {
				carrinho.Itens[i].Observacao = item.Observacao
			}
			return s.saveCarrinho(carrinho)
		}
	}

	// Adiciona novo item
	carrinho.Itens = append(carrinho.Itens, item)
	return s.saveCarrinho(carrinho)
}

// RemoverItem remove um item do carrinho
func (s *CarrinhoService) RemoverItem(ctx context.Context, clienteID, tenantID string, nome string, quantidade int) error {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return err
	}

	for i, item := range carrinho.Itens {
		if strings.EqualFold(item.Nome, nome) {
			if quantidade == 0 || quantidade >= item.Quantidade {
				// Remove o item completamente
				carrinho.Itens = append(carrinho.Itens[:i], carrinho.Itens[i+1:]...)
			} else {
				// Reduz a quantidade
				carrinho.Itens[i].Quantidade -= quantidade
			}
			return s.saveCarrinho(carrinho)
		}
	}

	return fmt.Errorf("item '%s' não encontrado no carrinho", nome)
}

// LimparCarrinho limpa todo o carrinho
func (s *CarrinhoService) LimparCarrinho(ctx context.Context, clienteID, tenantID string) error {
	key := s.getKey(clienteID, tenantID)
	return s.cache.Del(ctx, key).Err()
}

// CalcularTotal calcula o total do carrinho
func (s *CarrinhoService) CalcularTotal(carrinho *dto.Carrinho) float64 {
	total := 0.0
	for _, item := range carrinho.Itens {
		total += item.Preco * float64(item.Quantidade)
	}
	return total
}

// CalcularTempoEstimado calcula o tempo estimado do carrinho
func (s *CarrinhoService) CalcularTempoEstimado(carrinho *dto.Carrinho) int {
	if len(carrinho.Itens) == 0 {
		return 0
	}

	tempoBase := 15 // minutos
	tempoPorItem := 5

	totalItems := 0
	for _, item := range carrinho.Itens {
		totalItems += item.Quantidade
	}

	return tempoBase + (totalItems * tempoPorItem)
}

// FinalizarCarrinho finaliza o pedido
func (s *CarrinhoService) FinalizarCarrinho(ctx context.Context, clienteID, tenantID, clienteNome string) (*dto.PedidoConfirmado, error) {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return nil, err
	}

	if len(carrinho.Itens) == 0 {
		return nil, fmt.Errorf("carrinho vazio")
	}

	// Converte itens do carrinho para PedidoExtraido
	pedidoExtraido := &dto.PedidoExtraido{}
	for _, item := range carrinho.Itens {
		pedidoExtraido.Itens = append(pedidoExtraido.Itens, dto.ItemPedidoInput{
			Nome:          item.Nome,
			Quantidade:    item.Quantidade,
			Observacao:    item.Observacao,
			PrecoUnitario: item.Preco,
		})
	}

	// Processa o pedido
	pedidoConfirmado, err := s.pedidoService.ProcessarPedido(ctx, tenantID, clienteID, clienteNome, pedidoExtraido)
	if err != nil {
		return nil, err
	}

	// Limpa o carrinho
	if err := s.LimparCarrinho(ctx, clienteID, tenantID); err != nil {
		log.Printf("[Carrinho] Erro ao limpar carrinho após finalizar: %v", err)
	}

	return pedidoConfirmado, nil
}

// ============================================
// MÉTODOS PARA BUSCA HÍBRIDA
// ============================================

// ProcessarMensagem processa uma mensagem usando a abordagem híbrida
func (s *CarrinhoService) ProcessarMensagem(ctx context.Context, clienteID, tenantID, mensagem string) (*dto.Carrinho, error) {
	log.Printf("[Carrinho] Processando mensagem de %s: %s", clienteID, mensagem)

	// 1. Pré-processamento
	pp := s.preprocessor.Process(mensagem)
	log.Printf("[Carrinho] Mensagem limpa: %s", pp.Cleaned)

	// 2. LLM extrai intenção com mensagem limpa
	intencao, err := s.llmClient.ExtractIntent(pp.Cleaned, []dto.ProdutoItem{})
	if err != nil {
		return nil, fmt.Errorf("erro ao extrair intenção: %w", err)
	}

	log.Printf("[Carrinho] Intenção: %s, %d itens", intencao.Acao, len(intencao.Itens))

	// Se não for adicionar, não precisa buscar produtos
	if intencao.Acao != "adicionar" && intencao.Acao != "add" {
		return s.GetCarrinho(ctx, clienteID, tenantID)
	}

	// 3. Busca produtos no MySQL
	nomesProdutos := make([]string, len(intencao.Itens))
	for i, item := range intencao.Itens {
		nomesProdutos[i] = item.Nome
	}

	produtosEncontrados, err := s.produtoRepo.BuscarProdutosLote(ctx, tenantID, nomesProdutos)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar produtos: %w", err)
	}

	// 4. Separa encontrados e não encontrados
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
			log.Printf("[Carrinho] Produto encontrado: %s -> %s (R$ %.2f)",
				item.Nome, produto.Nome, produto.Preco)
		} else {
			naoEncontrados = append(naoEncontrados, item.Nome)
			log.Printf("[Carrinho] Produto não encontrado: %s", item.Nome)
		}
	}

	// 5. Se tiver não encontrados, tenta corrigir (segunda chamada LLM - Híbrida)
	if len(naoEncontrados) > 0 && len(produtosEncontrados) > 0 {
		log.Printf("[Carrinho] Tentando corrigir %d produtos não encontrados", len(naoEncontrados))

		// Busca produtos similares no banco
		similares, err := s.produtoRepo.BuscarProdutosLote(ctx, tenantID, naoEncontrados)
		if err != nil {
			log.Printf("[Carrinho] Erro ao buscar similares: %v", err)
		}

		// Se encontrou similares, pede LLM para corrigir
		if len(similares) > 0 {
			corrigidos, err := s.llmClient.CorrigirNomes(naoEncontrados, similares)
			if err != nil {
				log.Printf("[Carrinho] Erro ao corrigir nomes: %v", err)
			} else {
				// Adiciona os corrigidos ao carrinho
				for _, item := range corrigidos {
					if produto, ok := similares[item.Nome]; ok {
						encontrados = append(encontrados, dto.ItemCarrinho{
							Nome:       produto.Nome,
							Quantidade: item.Quantidade,
							Observacao: item.Observacao,
							Preco:      produto.Preco,
						})
						log.Printf("[Carrinho] Produto corrigido: %s -> %s", item.Nome, produto.Nome)
					}
				}
			}
		}
	}

	// 6. Adiciona todos os produtos encontrados ao carrinho
	_, err = s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return nil, err
	}

	for _, item := range encontrados {
		if err := s.AdicionarItem(ctx, clienteID, tenantID, item); err != nil {
			log.Printf("[Carrinho] Erro ao adicionar item %s: %v", item.Nome, err)
		}
	}

	// 7. Retorna o carrinho atualizado
	return s.GetCarrinho(ctx, clienteID, tenantID)
}

// BuscarProdutos busca produtos por nome no MySQL
func (s *CarrinhoService) BuscarProdutos(ctx context.Context, tenantID, termo string, limit int) ([]dto.ProdutoItem, error) {
	return s.produtoRepo.BuscarProdutosPorNome(ctx, tenantID, termo, limit)
}

// BuscarProdutosLote busca múltiplos produtos de uma vez
func (s *CarrinhoService) BuscarProdutosLote(ctx context.Context, tenantID string, nomes []string) (map[string]dto.ProdutoItem, error) {
	return s.produtoRepo.BuscarProdutosLote(ctx, tenantID, nomes)
}
