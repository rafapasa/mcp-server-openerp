// internal/service/cardapio_service.go - CLEAN mantendo métodos públicos do front
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/cache"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"go.uber.org/zap"
)

type cardapioService struct {
	produtoRepo repository.ProdutoRepository
	tenantRepo  repository.TenantRepository
	cache       *cache.Cache
}

func NewCardapioService(
	produtoRepo repository.ProdutoRepository,
	tenantRepo repository.TenantRepository,
	cache *cache.Cache,
) CardapioServiceInterface {
	return &cardapioService{
		produtoRepo: produtoRepo,
		tenantRepo:  tenantRepo,
		cache:       cache,
	}
}

// GetCardapio - com cache 1h via cache.Cache
func (s *cardapioService) GetCardapio(ctx context.Context, tenantID uint) ([]dto.ProdutoItem, error) {
	cacheKey := fmt.Sprintf("cardapio:%d", tenantID)

	cardapio, err := cache.GetOrSet(s.cache, ctx, cacheKey, 1*time.Hour, func() ([]dto.ProdutoItem, error) {
		produtos, err := s.produtoRepo.FindByTenantDisponiveis(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar cardápio: %w", err)
		}
		var out []dto.ProdutoItem
		for _, p := range produtos {
			cat := ""
			if p.Categoria != nil {
				cat = p.Categoria.Nome
			}
			out = append(out, dto.ProdutoItem{
				ID:           p.ID,
				Nome:         p.Nome,
				Categoria:    cat,
				Descricao:    p.Descricao,
				Preco:        p.Preco,
				Ingredientes: p.Ingredientes,
				Disponivel:   p.Disponivel,
			})
		}
		return out, nil
	})
	if err != nil {
		logger.Error(ctx, "Erro GetCardapio", zap.Error(err), zap.Uint("tenant_id", tenantID))
		return nil, err
	}
	return cardapio, nil
}

// BuscarProdutoPorNome - mantém assinatura original com string (front usa string), mas aceita uint também internamente
func (s *cardapioService) BuscarProdutoPorNome(ctx context.Context, tenantID string, nome string) (*dto.ProdutoItem, error) {
	var tenantIDUint uint
	if _, err := fmt.Sscan(tenantID, &tenantIDUint); err != nil {
		// tenta parse direto se já for número
		return nil, fmt.Errorf("tenant_id inválido: %w", err)
	}
	produto, err := s.produtoRepo.FindByNome(ctx, tenantIDUint, nome)
	if err != nil {
		return nil, err
	}
	cat := ""
	if produto.Categoria != nil {
		cat = produto.Categoria.Nome
	}
	return &dto.ProdutoItem{
		ID:           produto.ID,
		Nome:         produto.Nome,
		Categoria:    cat,
		Descricao:    produto.Descricao,
		Preco:        produto.Preco,
		Ingredientes: produto.Ingredientes,
		Disponivel:   produto.Disponivel,
	}, nil
}

// ItemExisteNoCardapio - mantém pro front e pro pedido_service legado, agora sem fuzzy agressivo
func (s *cardapioService) ItemExisteNoCardapio(cardapio []dto.ProdutoItem, nome string) (*dto.ProdutoItem, error) {
	nomeLower := strings.ToLower(strings.TrimSpace(nome))
	for _, item := range cardapio {
		if strings.ToLower(item.Nome) == nomeLower {
			return &item, nil
		}
		if strings.Contains(strings.ToLower(item.Nome), nomeLower) ||
			strings.Contains(nomeLower, strings.ToLower(item.Nome)) {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("Produto %s não localizado", nome)
}

// EncontrarItemSimilar - mantém pro front
func (s *cardapioService) EncontrarItemSimilar(cardapio []dto.ProdutoItem, nome string) string {
	nomeLower := strings.ToLower(strings.TrimSpace(nome))
	bestMatch := ""
	bestScore := 0
	for _, item := range cardapio {
		score := similarityScore(nomeLower, strings.ToLower(item.Nome))
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

// FormatarCardapio - mantém pro front
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

func (s *cardapioService) FindByID(ctx context.Context, id uint) (*dto.ProdutoDTO, error) {
	produto, err := s.produtoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	catNome := ""
	if produto.Categoria != nil {
		catNome = produto.Categoria.Nome
	}
	return &dto.ProdutoDTO{
		ID:            produto.ID,
		TenantID:      produto.TenantID,
		CategoriaID:   produto.CategoriaID,
		CategoriaNome: catNome,
		Nome:          produto.Nome,
		Descricao:     produto.Descricao,
		Preco:         produto.Preco,
		Disponivel:    produto.Disponivel,
		CreatedAt:     produto.CreatedAt,
		UpdatedAt:     produto.UpdatedAt,
	}, nil
}

func (s *cardapioService) ListWithFilters(ctx context.Context, tenantID uint, categoriaID *uint, disponivel *bool, nome string, page, limit int) ([]dto.ProdutoDTO, int64, error) {
	offset := (page - 1) * limit
	produtos, total, err := s.produtoRepo.FindWithFilters(ctx, tenantID, categoriaID, disponivel, nome, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]dto.ProdutoDTO, len(produtos))
	for i, p := range produtos {
		catNome := ""
		if p.Categoria != nil {
			catNome = p.Categoria.Nome
		}
		result[i] = dto.ProdutoDTO{
			ID:            p.ID,
			TenantID:      p.TenantID,
			CategoriaID:   p.CategoriaID,
			CategoriaNome: catNome,
			Nome:          p.Nome,
			Descricao:     p.Descricao,
			Preco:         p.Preco,
			Disponivel:    p.Disponivel,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
		}
	}
	return result, total, nil
}
