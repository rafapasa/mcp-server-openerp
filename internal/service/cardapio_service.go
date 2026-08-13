// internal/service/cardapio_service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// cardapioService gerencia as operações do cardápio
type cardapioService struct {
	produtoRepo repository.ProdutoRepository
	tenantRepo  repository.TenantRepository
	cache       *redis.Client
}

func NewCardapioService(
	produtoRepo repository.ProdutoRepository,
	tenantRepo repository.TenantRepository,
	cache *redis.Client,
) CardapioServiceInterface {
	return &cardapioService{
		produtoRepo: produtoRepo,
		tenantRepo:  tenantRepo,
		cache:       cache,
	}
}

// GetCardapio busca o cardápio do restaurante (com cache)
func (s *cardapioService) GetCardapio(ctx context.Context, tenantID uint) ([]dto.ProdutoItem, error) {
	// 1. Tenta buscar do cache
	cacheKey := fmt.Sprintf("cardapio:%d", tenantID)

	cached, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var cardapio []dto.ProdutoItem
		if err := json.Unmarshal([]byte(cached), &cardapio); err == nil {
			logger.Debug(ctx, "Cardápio encontrado no cache",
				zap.Uint("tenant_id", tenantID),
				zap.Int("itens", len(cardapio)),
			)
			return cardapio, nil
		}
	}

	// 2. Busca produtos do banco
	produtos, err := s.produtoRepo.FindByTenantDisponiveis(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, "Erro ao buscar cardápio no banco",
			zap.Error(err),
			zap.Uint("tenant_id", tenantID),
		)
		return nil, fmt.Errorf("erro ao buscar cardápio: %w", err)
	}

	// 3. Converte para ProdutoItem
	var cardapio []dto.ProdutoItem
	for _, p := range produtos {
		categoria := ""
		if p.Categoria != nil {
			categoria = p.Categoria.Nome
		}

		cardapio = append(cardapio, dto.ProdutoItem{
			ID:           p.ID,
			Nome:         p.Nome,
			Categoria:    categoria,
			Descricao:    p.Descricao,
			Preco:        p.Preco,
			Ingredientes: p.Ingredientes,
			Disponivel:   p.Disponivel,
		})
	}

	// 4. Salva no cache
	if len(cardapio) > 0 {
		data, _ := json.Marshal(cardapio)
		s.cache.Set(ctx, cacheKey, data, time.Hour)
		logger.Info(ctx, "Cardápio cacheado",
			zap.Uint("tenant_id", tenantID),
			zap.Int("itens", len(cardapio)),
		)
	}

	return cardapio, nil
}

// BuscarProdutoPorNome busca um produto pelo nome (case insensitive)
func (s *cardapioService) BuscarProdutoPorNome(ctx context.Context, tenantID string, nome string) (*dto.ProdutoItem, error) {
	var tenantIDUint uint
	if _, err := fmt.Sscan(tenantID, &tenantIDUint); err != nil {
		return nil, fmt.Errorf("tenant_id inválido: %w", err)
	}

	produto, err := s.produtoRepo.FindByNome(ctx, tenantIDUint, nome)
	if err != nil {
		logger.Warn(ctx, "Produto não encontrado pelo nome",
			zap.String("nome", nome),
			zap.Uint("tenant_id", tenantIDUint),
		)
		return nil, err
	}

	categoria := ""
	if produto.Categoria != nil {
		categoria = produto.Categoria.Nome
	}

	return &dto.ProdutoItem{
		ID:           produto.ID,
		Nome:         produto.Nome,
		Categoria:    categoria,
		Descricao:    produto.Descricao,
		Preco:        produto.Preco,
		Ingredientes: produto.Ingredientes,
		Disponivel:   produto.Disponivel,
	}, nil
}

// ItemExisteNoCardapio verifica se um item existe e retorna seu preço
func (s *cardapioService) ItemExisteNoCardapio(cardapio []dto.ProdutoItem, nome string) (bool, float64) {
	nomeLower := strings.ToLower(strings.TrimSpace(nome))

	for _, item := range cardapio {
		if strings.ToLower(item.Nome) == nomeLower {
			return true, item.Preco
		}

		if strings.Contains(strings.ToLower(item.Nome), nomeLower) ||
			strings.Contains(nomeLower, strings.ToLower(item.Nome)) {
			return true, item.Preco
		}
	}

	return false, 0
}

// EncontrarItemSimilar tenta encontrar um item similar no cardápio
func (s *cardapioService) EncontrarItemSimilar(cardapio []dto.ProdutoItem, nome string) string {
	nomeLower := strings.ToLower(strings.TrimSpace(nome))
	bestMatch := ""
	bestScore := 0

	for _, item := range cardapio {
		itemLower := strings.ToLower(item.Nome)
		score := similarityScore(nomeLower, itemLower)
		if score > bestScore {
			bestScore = score
			bestMatch = item.Nome
		}
	}

	if bestScore > 3 {
		return bestMatch
	}
	return ""
}

// similarityScore calcula uma pontuação de similaridade simples
func similarityScore(a, b string) int {
	score := 0
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	for _, wordA := range wordsA {
		for _, wordB := range wordsB {
			if len(wordA) > 3 && len(wordB) > 3 {
				if strings.Contains(wordA, wordB) || strings.Contains(wordB, wordA) {
					score += 2
				}
			}
		}
	}

	return score
}

// FormatarCardapio formata o cardápio para enviar no prompt da IA
func (s *cardapioService) FormatarCardapio(cardapio []dto.ProdutoItem) string {
	var sb strings.Builder
	sb.WriteString("CARDÁPIO:\n")

	categoriaAtual := ""
	for _, item := range cardapio {
		if item.Categoria != categoriaAtual {
			categoriaAtual = item.Categoria
			sb.WriteString(fmt.Sprintf("\n--- %s ---\n", item.Categoria))
		}
		sb.WriteString(fmt.Sprintf("- %s: R$ %.2f", item.Nome, item.Preco))
		if item.Descricao != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", item.Descricao))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ============================================
// LIST METHODS
// ============================================

// ListWithFilters lista produtos com filtros e paginação
func (s *cardapioService) ListWithFilters(ctx context.Context, tenantID uint, categoriaID *uint, disponivel *bool, nome string, page, limit int) ([]dto.ProdutoDTO, int64, error) {
	offset := (page - 1) * limit

	produtos, total, err := s.produtoRepo.FindWithFilters(ctx, tenantID, categoriaID, disponivel, nome, limit, offset)
	if err != nil {
		logger.Error(ctx, "Erro ao listar produtos com filtros",
			zap.Error(err),
			zap.Uint("tenant_id", tenantID),
		)
		return nil, 0, err
	}

	result := make([]dto.ProdutoDTO, len(produtos))
	for i, p := range produtos {
		categoriaNome := ""
		if p.Categoria != nil {
			categoriaNome = p.Categoria.Nome
		}

		result[i] = dto.ProdutoDTO{
			ID:            p.ID,
			TenantID:      p.TenantID,
			CategoriaID:   p.CategoriaID,
			CategoriaNome: categoriaNome,
			Nome:          p.Nome,
			Descricao:     p.Descricao,
			Preco:         p.Preco,
			Disponivel:    p.Disponivel,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
		}
	}

	logger.Debug(ctx, "Produtos listados com filtros",
		zap.Uint("tenant_id", tenantID),
		zap.Int("total", int(total)),
		zap.Int("page", page),
		zap.Int("limit", limit),
	)

	return result, total, nil
}

// FindByID busca um produto por ID
func (s *cardapioService) FindByID(ctx context.Context, id uint) (*dto.ProdutoDTO, error) {
	produto, err := s.produtoRepo.FindByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "Produto não encontrado", zap.Uint("id", id), zap.Error(err))
		return nil, err
	}

	categoriaNome := ""
	if produto.Categoria != nil {
		categoriaNome = produto.Categoria.Nome
	}

	return &dto.ProdutoDTO{
		ID:            produto.ID,
		TenantID:      produto.TenantID,
		CategoriaID:   produto.CategoriaID,
		CategoriaNome: categoriaNome,
		Nome:          produto.Nome,
		Descricao:     produto.Descricao,
		Preco:         produto.Preco,
		Disponivel:    produto.Disponivel,
		CreatedAt:     produto.CreatedAt,
		UpdatedAt:     produto.UpdatedAt,
	}, nil
}
