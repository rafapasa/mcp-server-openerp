// internal/service/cardapio_service_test.go
package service

import (
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/mocks"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// novoCardapioServiceMock cria o cardapioService com repositório mockado.
// cache e tenantRepo são nil (GetOrSet com cache nil executa o fetch direto;
// tenantRepo não é usado pelos métodos do cardapioService).
func novoCardapioServiceMock(t *testing.T) (*cardapioService, *mocks.MockProdutoRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	produtoRepo := mocks.NewMockProdutoRepository(ctrl)

	svc := &cardapioService{
		produtoRepo: produtoRepo,
		tenantRepo:  nil,
		cache:       nil,
	}
	return svc, produtoRepo
}

func TestCardapioService_GetCardapio(t *testing.T) {
	catID := uint(10)

	t.Run("sucesso: converte produtos com categoria", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtos := []models.Produto{
			{ID: 1, TenantID: 1, Nome: "X-Bacon", Preco: 25.0, Disponivel: true, Categoria: &models.Categoria{ID: catID, Nome: "Lanches"}},
			{ID: 2, TenantID: 1, Nome: "Coca", Preco: 8.0, Disponivel: true, Categoria: &models.Categoria{ID: 3, Nome: "Bebidas"}},
		}
		produtoRepo.EXPECT().FindByTenantDisponiveis(testCtx(), uint(1)).Return(produtos, nil)

		cardapio, err := svc.GetCardapio(testCtx(), 1)
		require.NoError(t, err)
		require.Len(t, cardapio, 2)
		assert.Equal(t, uint(1), cardapio[0].ID)
		assert.Equal(t, "X-Bacon", cardapio[0].Nome)
		assert.Equal(t, "Lanches", cardapio[0].Categoria)
		assert.Equal(t, 25.0, cardapio[0].Preco)
		assert.Equal(t, "Bebidas", cardapio[1].Categoria)
	})

	t.Run("sucesso: produto sem categoria fica com categoria vazia", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtos := []models.Produto{{ID: 1, TenantID: 1, Nome: "Produto Solto", Preco: 5.0}}
		produtoRepo.EXPECT().FindByTenantDisponiveis(testCtx(), uint(1)).Return(produtos, nil)

		cardapio, err := svc.GetCardapio(testCtx(), 1)
		require.NoError(t, err)
		require.Len(t, cardapio, 1)
		assert.Equal(t, "", cardapio[0].Categoria)
	})

	t.Run("erro: repositório falha e propaga", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtoRepo.EXPECT().FindByTenantDisponiveis(testCtx(), uint(1)).Return(nil, assert.AnError)

		cardapio, err := svc.GetCardapio(testCtx(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "erro ao buscar cardápio")
		assert.Nil(t, cardapio)
	})
}

func TestCardapioService_BuscarProdutoPorNome(t *testing.T) {
	t.Run("erro: tenant_id não numérico", func(t *testing.T) {
		svc, _ := novoCardapioServiceMock(t)
		resultado, err := svc.BuscarProdutoPorNome(testCtx(), "abc", "X-Bacon")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_id inválido")
		assert.Nil(t, resultado)
	})

	t.Run("erro: repositório falha e propaga com wrap", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtoRepo.EXPECT().FindByNome(testCtx(), uint(1), "X-Bacon").Return(nil, assert.AnError)

		resultado, err := svc.BuscarProdutoPorNome(testCtx(), "1", "X-Bacon")
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "erro ao buscar produto por nome")
		assert.Nil(t, resultado)
	})

	t.Run("sucesso: converte com categoria", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produto := &models.Produto{ID: 1, TenantID: 1, Nome: "X-Bacon", Preco: 25.0, Categoria: &models.Categoria{ID: 10, Nome: "Lanches"}}
		produtoRepo.EXPECT().FindByNome(testCtx(), uint(1), "X-Bacon").Return(produto, nil)

		resultado, err := svc.BuscarProdutoPorNome(testCtx(), "1", "X-Bacon")
		require.NoError(t, err)
		assert.Equal(t, uint(1), resultado.ID)
		assert.Equal(t, "Lanches", resultado.Categoria)
		assert.Equal(t, 25.0, resultado.Preco)
	})
}

func TestCardapioService_FindByID(t *testing.T) {
	t.Run("sucesso: converte com categoria", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produto := &models.Produto{ID: 1, TenantID: 1, Nome: "X-Bacon", Preco: 25.0, Categoria: &models.Categoria{ID: 10, Nome: "Lanches"}}
		produtoRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(produto, nil)

		resultado, err := svc.FindByID(testCtx(), 1)
		require.NoError(t, err)
		assert.Equal(t, uint(1), resultado.ID)
		assert.Equal(t, "Lanches", resultado.CategoriaNome)
		assert.Equal(t, 25.0, resultado.Preco)
	})

	t.Run("sucesso: sem categoria", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produto := &models.Produto{ID: 1, TenantID: 1, Nome: "X-Bacon", Preco: 25.0}
		produtoRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(produto, nil)

		resultado, err := svc.FindByID(testCtx(), 1)
		require.NoError(t, err)
		assert.Equal(t, "", resultado.CategoriaNome)
	})

	t.Run("erro: repositório falha e propaga com wrap", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtoRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(nil, assert.AnError)

		resultado, err := svc.FindByID(testCtx(), 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "erro ao buscar produto")
		assert.Nil(t, resultado)
	})
}

func TestCardapioService_ListWithFilters(t *testing.T) {
	catID := uint(10)

	t.Run("sucesso: converte lista com categoria e total", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtos := []models.Produto{
			{ID: 1, TenantID: 1, Nome: "X-Bacon", Preco: 25.0, CategoriaID: &catID, Categoria: &models.Categoria{ID: catID, Nome: "Lanches"}},
		}
		produtoRepo.EXPECT().FindWithFilters(testCtx(), uint(1), gomock.Any(), gomock.Any(), "bacon", 10, 0).Return(produtos, int64(1), nil)

		resultado, total, err := svc.ListWithFilters(testCtx(), 1, &catID, nil, "bacon", 1, 10)
		require.NoError(t, err)
		require.Len(t, resultado, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Lanches", resultado[0].CategoriaNome)
		assert.Equal(t, uint(1), resultado[0].ID)
	})

	t.Run("erro: repositório falha e propaga com wrap", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtoRepo.EXPECT().FindWithFilters(testCtx(), uint(1), nil, nil, "", 10, 0).Return(nil, int64(0), assert.AnError)

		_, _, err := svc.ListWithFilters(testCtx(), 1, nil, nil, "", 1, 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "erro ao listar produtos")
	})
}

func TestCardapioService_ReduzirPorKeywords(t *testing.T) {
	t.Run("keywords vazio: retorna slice vazio sem consultar repositório", func(t *testing.T) {
		svc, _ := novoCardapioServiceMock(t)
		resultado, err := svc.ReduzirPorKeywords(testCtx(), 1, []llm.LLMKeywordItemResult{})
		require.NoError(t, err)
		assert.NotNil(t, resultado)
		assert.Empty(t, resultado)
	})

	t.Run("sucesso: filtra por nome e unidade com dedupe", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtos := []models.Produto{
			{ID: 1, TenantID: 1, Nome: "X-Bacon Completo", Preco: 25.0},
			{ID: 2, TenantID: 1, Nome: "Coca-Cola Lata", Preco: 8.0},
			{ID: 3, TenantID: 1, Nome: "Suco de Laranja", Preco: 10.0},
		}
		produtoRepo.EXPECT().FindByTenantDisponiveis(testCtx(), uint(1)).Return(produtos, nil)

		keywords := []llm.LLMKeywordItemResult{
			{Nome: "x-bacon"},
			{Nome: "coca"},
			{Nome: "x-bacon"}, // repetido: deve deduplicar por ID
		}
		resultado, err := svc.ReduzirPorKeywords(testCtx(), 1, keywords)
		require.NoError(t, err)
		require.Len(t, resultado, 2)
		assert.Equal(t, uint(1), resultado[0].ID)
		assert.Equal(t, uint(2), resultado[1].ID)
	})

	t.Run("sucesso: fallback por contains quando unidade não casa no 1º loop", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtos := []models.Produto{
			{ID: 1, TenantID: 1, Nome: "X-Bacon", Preco: 25.0},
		}
		produtoRepo.EXPECT().FindByTenantDisponiveis(testCtx(), uint(1)).Return(produtos, nil)

		// unidade "coca" não existe em "x-bacon": o 1º loop falha e o fallback entra
		resultado, err := svc.ReduzirPorKeywords(testCtx(), 1, []llm.LLMKeywordItemResult{{Nome: "bacon", Unidade: "coca"}})
		require.NoError(t, err)
		require.Len(t, resultado, 1)
		assert.Equal(t, uint(1), resultado[0].ID)
	})

	t.Run("erro: GetCardapio falha e propaga", func(t *testing.T) {
		svc, produtoRepo := novoCardapioServiceMock(t)
		produtoRepo.EXPECT().FindByTenantDisponiveis(testCtx(), uint(1)).Return(nil, assert.AnError)

		resultado, err := svc.ReduzirPorKeywords(testCtx(), 1, []llm.LLMKeywordItemResult{{Nome: "x-bacon"}})
		require.Error(t, err)
		assert.Nil(t, resultado)
	})
}

func TestCardapioService_ItemExisteNoCardapio(t *testing.T) {
	cardapio := []dto.ProdutoItem{
		{Nome: "X-Bacon", Preco: 25.0, Categoria: "Lanches"},
		{Nome: "Coca-Cola", Preco: 8.0, Categoria: "Bebidas"},
	}
	svc, _ := novoCardapioServiceMock(t)

	t.Run("match exato case-insensitive", func(t *testing.T) {
		item, err := svc.ItemExisteNoCardapio(cardapio, "x-bacon")
		require.NoError(t, err)
		assert.Equal(t, "X-Bacon", item.Nome)
	})

	t.Run("match parcial (nome contém item)", func(t *testing.T) {
		item, err := svc.ItemExisteNoCardapio(cardapio, "bacon")
		require.NoError(t, err)
		assert.Equal(t, "X-Bacon", item.Nome)
	})

	t.Run("não encontrado: retorna erro", func(t *testing.T) {
		item, err := svc.ItemExisteNoCardapio(cardapio, "pizza")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pizza")
		assert.Nil(t, item)
	})

	t.Run("nome vazio: retorna erro de validação", func(t *testing.T) {
		item, err := svc.ItemExisteNoCardapio(cardapio, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "não pode ser vazio")
		assert.Nil(t, item)
	})
}

func TestCardapioService_EncontrarItemSimilar(t *testing.T) {
	cardapio := []dto.ProdutoItem{
		{Nome: "X-Bacon Cheddar"},
		{Nome: "Coca-Cola Lata"},
	}
	svc, _ := novoCardapioServiceMock(t)

	t.Run("match por similaridade (score > 3)", func(t *testing.T) {
		resultado := svc.EncontrarItemSimilar(cardapio, "x-bacon cheddar")
		assert.Equal(t, "X-Bacon Cheddar", resultado)
	})

	t.Run("sem match: retorna string vazia", func(t *testing.T) {
		resultado := svc.EncontrarItemSimilar(cardapio, "pizza calabresa")
		assert.Equal(t, "", resultado)
	})
}

func TestCardapioService_similarityScore(t *testing.T) {
	t.Run("palavras que se contêm somam 2 por par", func(t *testing.T) {
		score := similarityScore("x-bacon cheddar", "x-bacon cheddar")
		assert.Equal(t, 4, score)
	})

	t.Run("sem sobreposição: 0", func(t *testing.T) {
		score := similarityScore("pizza", "bebida")
		assert.Equal(t, 0, score)
	})

	t.Run("palavras curtas (<=3) são ignoradas", func(t *testing.T) {
		score := similarityScore("abc", "abc")
		assert.Equal(t, 0, score)
	})
}

func TestCardapioService_BuscarProdutoPorIdNoCardapio(t *testing.T) {
	cardapio := []dto.ProdutoItem{
		{ID: 1, Nome: "X-Bacon"},
		{ID: 2, Nome: "Coca-Cola"},
	}
	svc, _ := novoCardapioServiceMock(t)

	t.Run("cardápio vazio", func(t *testing.T) {
		item, err := svc.BuscarProdutoPorIdNoCardapio([]dto.ProdutoItem{}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cardápio vazio")
		assert.Nil(t, item)
	})

	t.Run("produtoID zero", func(t *testing.T) {
		item, err := svc.BuscarProdutoPorIdNoCardapio(cardapio, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "produtoID inválido")
		assert.Nil(t, item)
	})

	t.Run("encontrado", func(t *testing.T) {
		item, err := svc.BuscarProdutoPorIdNoCardapio(cardapio, 2)
		require.NoError(t, err)
		assert.Equal(t, "Coca-Cola", item.Nome)
	})

	t.Run("não encontrado", func(t *testing.T) {
		item, err := svc.BuscarProdutoPorIdNoCardapio(cardapio, 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "99")
		assert.Nil(t, item)
	})
}

func TestCardapioService_FormatarCardapio(t *testing.T) {
	svc, _ := novoCardapioServiceMock(t)

	t.Run("sem itens", func(t *testing.T) {
		assert.Equal(t, "CARDÁPIO:\n", svc.FormatarCardapio([]dto.ProdutoItem{}))
	})

	t.Run("agrupa por categoria e formata preço", func(t *testing.T) {
		cardapio := []dto.ProdutoItem{
			{Nome: "X-Bacon", Categoria: "Lanches", Preco: 25.0, Descricao: "com bacon"},
			{Nome: "Coca", Categoria: "Bebidas", Preco: 8.5},
		}
		esperado := "CARDÁPIO:\n\n--- Lanches ---\n- X-Bacon: R$ 25.00 (com bacon)\n\n--- Bebidas ---\n- Coca: R$ 8.50\n"
		assert.Equal(t, esperado, svc.FormatarCardapio(cardapio))
	})
}

func TestCardapioService_ListarCategoriasHumanizado(t *testing.T) {
	svc, _ := novoCardapioServiceMock(t)

	t.Run("categorias únicas, sem duplicatas e sem vazias", func(t *testing.T) {
		cardapio := []dto.ProdutoItem{
			{Categoria: "Lanches"},
			{Categoria: "Bebidas"},
			{Categoria: "Lanches"},
			{Categoria: "  "},
			{Categoria: "Sobremesas"},
		}
		resultado := svc.ListarCategoriasHumanizado(cardapio)
		assert.Equal(t, "Temos Lanches, Bebidas, Sobremesas. Você tá com fome, com sede ou os dois? 😊", resultado)
	})

	t.Run("sem categorias: mensagem padrão", func(t *testing.T) {
		resultado := svc.ListarCategoriasHumanizado([]dto.ProdutoItem{{Categoria: ""}, {Categoria: "  "}})
		assert.Equal(t, "No momento não encontrei categorias no cardápio. Me diga o que você procura! 😊", resultado)
	})
}

func TestCardapioService_ListarProdutosHumanizado(t *testing.T) {
	svc, _ := novoCardapioServiceMock(t)

	t.Run("sem filtro: 3 por categoria e sem categoria vira Outros", func(t *testing.T) {
		cardapio := []dto.ProdutoItem{
			{Nome: "X-Bacon", Categoria: "Lanches", Preco: 25.0},
			{Nome: "X-Salada", Categoria: "Lanches", Preco: 20.0},
			{Nome: "X-Tudo", Categoria: "Lanches", Preco: 30.0},
			{Nome: "X-Extra", Categoria: "Lanches", Preco: 35.0}, // 4º: fica de fora
			{Nome: "Coca", Categoria: "Bebidas", Preco: 8.0},
			{Nome: "Solto", Preco: 5.0}, // sem categoria → "Outros"
		}
		esperado := "Aqui vai um gostinho do cardápio:\n" +
			"\n*Lanches*\n• X-Bacon — R$ 25.00\n• X-Salada — R$ 20.00\n• X-Tudo — R$ 30.00\n" +
			"\n*Bebidas*\n• Coca — R$ 8.00\n" +
			"\n*Outros*\n• Solto — R$ 5.00\n" +
			"\nTem muito mais além desses. Me fala uma categoria ou o nome do produto! 😊"
		assert.Equal(t, esperado, svc.ListarProdutosHumanizado(cardapio, ""))
	})

	t.Run("com filtro: match por nome", func(t *testing.T) {
		cardapio := []dto.ProdutoItem{
			{Nome: "X-Bacon", Categoria: "Lanches", Preco: 25.0},
			{Nome: "X-Salada", Categoria: "Lanches", Preco: 20.0},
			{Nome: "Coca", Categoria: "Bebidas", Preco: 8.0},
		}
		esperado := "Olha o que encontrei pra \"bacon\":\n• X-Bacon — R$ 25.00\n"
		assert.Equal(t, esperado, svc.ListarProdutosHumanizado(cardapio, "bacon"))
	})

	t.Run("com filtro sem match: mensagem", func(t *testing.T) {
		cardapio := []dto.ProdutoItem{{Nome: "X-Bacon", Categoria: "Lanches", Preco: 25.0}}
		assert.Equal(t, "Não achei produtos com \"pizza\". Quer ver as categorias? 😊", svc.ListarProdutosHumanizado(cardapio, "pizza"))
	})

	t.Run("com filtro com mais de 10 resultados: trunca e avisa", func(t *testing.T) {
		var cardapio []dto.ProdutoItem
		for i := 1; i <= 12; i++ {
			cardapio = append(cardapio, dto.ProdutoItem{Nome: "Item Bacon", Preco: float64(i)})
		}
		resultado := svc.ListarProdutosHumanizado(cardapio, "bacon")
		assert.Contains(t, resultado, "Tem mais além desses. Quer filtrar melhor?")
		assert.Equal(t, 10, strings.Count(resultado, "• "))
	})
}

func novoCardapioServiceMockWithCache(t *testing.T) (*cardapioService, *mocks.MockProdutoRepository, *mocks.MockRedisInterface) {
	t.Helper()
	ctrl := gomock.NewController(t)
	produtoRepo := mocks.NewMockProdutoRepository(ctrl)
	redisMock := mocks.NewMockRedisInterface(ctrl)

	svc := &cardapioService{
		produtoRepo: produtoRepo,
		tenantRepo:  nil,
		cache:       redisMock,
	}
	return svc, produtoRepo, redisMock
}

func TestCardapioService_InvalidateCache(t *testing.T) {
	t.Run("invalida chave principal e pattern", func(t *testing.T) {
		svc, _, redisMock := novoCardapioServiceMockWithCache(t)
		ctx := testCtx()
		redisMock.EXPECT().DeleteWithContext(ctx, "cardapio:2").Return(nil).Times(1)
		redisMock.EXPECT().InvalidateByTenant(ctx, "cardapio:2*").Return(nil).Times(1)

		err := svc.InvalidateCache(ctx, 2)
		require.NoError(t, err)
	})

	t.Run("cache nil não quebra", func(t *testing.T) {
		svc, _ := novoCardapioServiceMock(t)
		err := svc.InvalidateCache(testCtx(), 2)
		require.NoError(t, err)
	})
}

func TestCardapioService_Create(t *testing.T) {
	t.Run("cria e invalida cache", func(t *testing.T) {
		svc, produtoRepo, redisMock := novoCardapioServiceMockWithCache(t)
		ctx := testCtx()
		produto := &models.Produto{TenantID: 2, Nome: "Coca", Preco: 5}

		produtoRepo.EXPECT().Create(ctx, produto).Return(nil)
		redisMock.EXPECT().DeleteWithContext(ctx, "cardapio:2").Return(nil)
		redisMock.EXPECT().InvalidateByTenant(ctx, "cardapio:2*").Return(nil)

		err := svc.Create(ctx, produto)
		require.NoError(t, err)
	})
}

func TestCardapioService_Update(t *testing.T) {
	t.Run("atualiza e invalida cache", func(t *testing.T) {
		svc, produtoRepo, redisMock := novoCardapioServiceMockWithCache(t)
		ctx := testCtx()
		produto := &models.Produto{ID: 10, TenantID: 2, Nome: "X-Bacon", Preco: 30}

		produtoRepo.EXPECT().Update(ctx, produto).Return(nil)
		redisMock.EXPECT().DeleteWithContext(ctx, "cardapio:2").Return(nil)
		redisMock.EXPECT().InvalidateByTenant(ctx, "cardapio:2*").Return(nil)

		err := svc.Update(ctx, produto)
		require.NoError(t, err)
	})
}

func TestCardapioService_Delete(t *testing.T) {
	t.Run("deleta e invalida cache", func(t *testing.T) {
		svc, produtoRepo, redisMock := novoCardapioServiceMockWithCache(t)
		ctx := testCtx()

		produtoRepo.EXPECT().Delete(ctx, uint(99)).Return(nil)
		redisMock.EXPECT().DeleteWithContext(ctx, "cardapio:2").Return(nil)
		redisMock.EXPECT().InvalidateByTenant(ctx, "cardapio:2*").Return(nil)

		err := svc.Delete(ctx, 99, 2)
		require.NoError(t, err)
	})
}

func TestCardapioService_UpdateDisponibilidade(t *testing.T) {
	t.Run("atualiza disponibilidade e invalida", func(t *testing.T) {
		svc, produtoRepo, redisMock := novoCardapioServiceMockWithCache(t)
		ctx := testCtx()
		produto := &models.Produto{ID: 5, TenantID: 2, Nome: "X-Bacon", Disponivel: true}

		produtoRepo.EXPECT().FindByID(ctx, uint(5)).Return(produto, nil)
		produtoRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		redisMock.EXPECT().DeleteWithContext(ctx, "cardapio:2").Return(nil)
		redisMock.EXPECT().InvalidateByTenant(ctx, "cardapio:2*").Return(nil)

		err := svc.UpdateDisponibilidade(ctx, 5, 2, false)
		require.NoError(t, err)
	})

	t.Run("tenant diferente retorna erro", func(t *testing.T) {
		svc, produtoRepo, _ := novoCardapioServiceMockWithCache(t)
		ctx := testCtx()
		produto := &models.Produto{ID: 5, TenantID: 2}

		produtoRepo.EXPECT().FindByID(ctx, uint(5)).Return(produto, nil)

		err := svc.UpdateDisponibilidade(ctx, 5, 99, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "não pertence ao tenant")
	})
}
