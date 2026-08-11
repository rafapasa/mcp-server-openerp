// internal/service/cardapio_service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"github.com/redis/go-redis/v9"
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
func (s *cardapioService) GetCardapio(ctx context.Context, tenantID string) ([]dto.ProdutoItem, error) {
	// 1. Tenta buscar do cache
	cacheKey := fmt.Sprintf("cardapio:%s", tenantID)

	cached, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var cardapio []dto.ProdutoItem
		if err := json.Unmarshal([]byte(cached), &cardapio); err == nil {
			log.Printf("[Cache] Cardápio encontrado no cache para tenant %s", tenantID)
			return cardapio, nil
		}
	}

	// 2. Converte tenantID para uint
	var tenantIDUint uint
	if _, err := fmt.Sscan(tenantID, &tenantIDUint); err != nil {
		return nil, fmt.Errorf("tenant_id inválido: %w", err)
	}

	// 3. Busca produtos do banco
	produtos, err := s.produtoRepo.FindByTenantDisponiveis(ctx, tenantIDUint)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cardápio: %w", err)
	}

	// 4. Converte para ProdutoItem
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

	// 5. Salva no cache
	if len(cardapio) > 0 {
		data, _ := json.Marshal(cardapio)
		s.cache.Set(ctx, cacheKey, data, time.Hour)
		log.Printf("[Cache] Cardápio cacheado para tenant %s (%d itens)", tenantID, len(cardapio))
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
