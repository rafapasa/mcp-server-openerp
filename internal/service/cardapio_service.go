// internal/service/cardapio_service.go - CLEAN mantendo métodos públicos do front
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"go.uber.org/zap"
)

type cardapioService struct {
	produtoRepo repository.ProdutoRepository
	tenantRepo  repository.TenantRepository
	cache       database.RedisInterface
}

func NewCardapioService(
	produtoRepo repository.ProdutoRepository,
	tenantRepo repository.TenantRepository,
	cache database.RedisInterface,
) CardapioServiceInterface {
	return &cardapioService{
		produtoRepo: produtoRepo,
		tenantRepo:  tenantRepo,
		cache:       cache,
	}
}

func (s *cardapioService) GetCardapio(ctx context.Context, tenantID uint) ([]dto.ProdutoItem, error) {
	cacheKey := fmt.Sprintf("cardapio:%d", tenantID)

	cardapio, err := database.GetOrSet(s.cache, ctx, cacheKey, 1*time.Hour, func() ([]dto.ProdutoItem, error) {
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

func (s *cardapioService) BuscarProdutoPorNome(ctx context.Context, tenantID string, nome string) (*dto.ProdutoItem, error) {
	var tenantIDUint uint
	if _, err := fmt.Sscan(tenantID, &tenantIDUint); err != nil {
		return nil, fmt.Errorf("tenant_id inválido: %w", err)
	}
	produto, err := s.produtoRepo.FindByNome(ctx, tenantIDUint, nome)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar produto por nome: %w", err)
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

func (s *cardapioService) ItemExisteNoCardapio(cardapio []dto.ProdutoItem, nome string) (*dto.ProdutoItem, error) {
	nomeLower := strings.ToLower(strings.TrimSpace(nome))
	if nomeLower == "" {
		return nil, fmt.Errorf("nome do produto não pode ser vazio")
	}
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

// ListarCategoriasHumanizado - frase curta para WhatsApp
func (s *cardapioService) ListarCategoriasHumanizado(cardapio []dto.ProdutoItem) string {
	seen := make(map[string]bool)
	var cats []string
	for _, p := range cardapio {
		c := strings.TrimSpace(p.Categoria)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		cats = append(cats, c)
	}
	if len(cats) == 0 {
		return "No momento não encontrei categorias no cardápio. Me diga o que você procura! 😊"
	}
	return fmt.Sprintf(
		"Temos %s. Você tá com fome, com sede ou os dois? 😊",
		strings.Join(cats, ", "),
	)
}

// ListarProdutosHumanizado - sem filtro: 3 de cada categoria; com filtro: max 10
func (s *cardapioService) ListarProdutosHumanizado(cardapio []dto.ProdutoItem, filtro string) string {
	filtro = strings.ToLower(strings.TrimSpace(filtro))

	if filtro != "" {
		var matched []dto.ProdutoItem
		for _, p := range cardapio {
			nome := strings.ToLower(p.Nome)
			cat := strings.ToLower(p.Categoria)
			if strings.Contains(nome, filtro) || strings.Contains(cat, filtro) {
				matched = append(matched, p)
			}
		}
		if len(matched) == 0 {
			return fmt.Sprintf("Não achei produtos com \"%s\". Quer ver as categorias? 😊", filtro)
		}
		const max = 10
		if len(matched) > max {
			matched = matched[:max]
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Olha o que encontrei pra \"%s\":\n", filtro))
		for _, p := range matched {
			sb.WriteString(fmt.Sprintf("• %s — R$ %.2f\n", p.Nome, p.Preco))
		}
		if len(matched) == max {
			sb.WriteString("\nTem mais além desses. Quer filtrar melhor?")
		}
		return sb.String()
	}

	// sem filtro: 3 primeiros de cada categoria
	byCat := make(map[string][]dto.ProdutoItem)
	var order []string
	for _, p := range cardapio {
		c := p.Categoria
		if c == "" {
			c = "Outros"
		}
		if _, ok := byCat[c]; !ok {
			order = append(order, c)
		}
		if len(byCat[c]) < 3 {
			byCat[c] = append(byCat[c], p)
		}
	}

	var sb strings.Builder
	sb.WriteString("Aqui vai um gostinho do cardápio:\n")
	for _, c := range order {
		sb.WriteString(fmt.Sprintf("\n*%s*\n", c))
		for _, p := range byCat[c] {
			sb.WriteString(fmt.Sprintf("• %s — R$ %.2f\n", p.Nome, p.Preco))
		}
	}
	sb.WriteString("\nTem muito mais além desses. Me fala uma categoria ou o nome do produto! 😊")
	return sb.String()
}

func (s *cardapioService) FindByID(ctx context.Context, id uint) (*dto.ProdutoDTO, error) {
	produto, err := s.produtoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar produto: %w", err)
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
		return nil, 0, fmt.Errorf("erro ao listar produtos: %w", err)
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

func (s *cardapioService) BuscarProdutoPorIdNoCardapio(cardapio []dto.ProdutoItem, produtoID uint) (*dto.ProdutoItem, error) {
	if len(cardapio) == 0 {
		return nil, fmt.Errorf("cardápio vazio")
	}
	if produtoID == 0 {
		return nil, fmt.Errorf("produtoID inválido")
	}
	for i := range cardapio {
		if cardapio[i].ID == produtoID {
			return &cardapio[i], nil
		}
	}
	return nil, fmt.Errorf("produto ID %d não encontrado no cardápio", produtoID)
}

// ReduzirPorKeywords - fluxo cardápio grande (max 30 itens)
func (s *cardapioService) ReduzirPorKeywords(
	ctx context.Context,
	tenantID uint,
	keywords []llm.LLMKeywordItemResult,
) ([]dto.ProdutoItem, error) {
	if len(keywords) == 0 {
		return []dto.ProdutoItem{}, nil
	}

	cardapio, err := s.GetCardapio(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var out []dto.ProdutoItem
	seen := make(map[uint]bool)

	for _, kw := range keywords {
		nomeLower := strings.ToLower(strings.TrimSpace(kw.Nome))
		unidadeLower := strings.ToLower(strings.TrimSpace(kw.Unidade))

		for _, p := range cardapio {
			if seen[p.ID] {
				continue
			}
			pNome := strings.ToLower(p.Nome)
			matchNome := strings.Contains(pNome, nomeLower) || strings.Contains(nomeLower, pNome)
			matchUnidade := unidadeLower == "" || strings.Contains(pNome, unidadeLower)
			if matchNome && matchUnidade {
				out = append(out, p)
				seen[p.ID] = true
			}
		}
	}

	if len(out) == 0 {
		for _, kw := range keywords {
			nomeLower := strings.ToLower(strings.TrimSpace(kw.Nome))
			for _, p := range cardapio {
				if seen[p.ID] {
					continue
				}
				if strings.Contains(strings.ToLower(p.Nome), nomeLower) {
					out = append(out, p)
					seen[p.ID] = true
				}
			}
		}
	}

	const maxReduzido = 30
	if len(out) > maxReduzido {
		out = out[:maxReduzido]
	}
	return out, nil
}
