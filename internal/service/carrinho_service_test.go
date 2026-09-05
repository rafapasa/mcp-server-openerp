package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/mock/gomock"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/mocks"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// novoCarrinhoServiceMock cria o carrinhoService com todas as dependências mockadas.
func novoCarrinhoServiceMock(t *testing.T) (*carrinhoService, *mocks.MockRedisInterface, *mocks.MockCardapioServiceInterface, *mocks.MockPedidoServiceInterface, *mocks.MockClienteServiceInterface, *mocks.MockProdutoRepository, *mocks.MockLLMServiceInterface) {
	t.Helper()

	ctrl := gomock.NewController(t)
	redisMock := mocks.NewMockRedisInterface(ctrl)
	cardapioMock := mocks.NewMockCardapioServiceInterface(ctrl)
	pedidoMock := mocks.NewMockPedidoServiceInterface(ctrl)
	clienteMock := mocks.NewMockClienteServiceInterface(ctrl)
	produtoMock := mocks.NewMockProdutoRepository(ctrl)
	llmMock := mocks.NewMockLLMServiceInterface(ctrl)

	svc := &carrinhoService{
		cache:           redisMock,
		cardapioService: cardapioMock,
		pedidoService:   pedidoMock,
		clienteService:  clienteMock,
		produtoRepo:     produtoMock,
		llmService:      llmMock,
	}
	return svc, redisMock, cardapioMock, pedidoMock, clienteMock, produtoMock, llmMock
}

// mockRedisCacheMiss configura o Redis para cache miss (GetOrSet cria carrinho novo).
func mockRedisCacheMiss(redisMock *mocks.MockRedisInterface) {
	redisMock.EXPECT().GetClient().Return(nil).AnyTimes()
}

// mockRedisCacheHit configura o Redis para devolver um carrinho cacheado.
func mockRedisCacheHit(t *testing.T, redisMock *mocks.MockRedisInterface, carrinho dto.Carrinho) {
	t.Helper()
	redisMock.EXPECT().GetClient().Return(redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})).AnyTimes()
	redisMock.EXPECT().GetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, dest any) error {
			c := dest.(**dto.Carrinho)
			*c = &carrinho
			return nil
		}).AnyTimes()
}

func TestCarrinhoService_getKey(t *testing.T) {
	svc := &carrinhoService{}
	assert.Equal(t, "carrinho:5:3", svc.getKey(3, 5))
}

func Test_parseUint(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want uint
	}{
		{name: "número válido", in: "123", want: 123},
		{name: "texto inválido", in: "abc", want: 0},
		{name: "vazio", in: "", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseUint(tt.in))
		})
	}
}

func Test_mustFormat(t *testing.T) {
	assert.Equal(t, "resumo", mustFormat("resumo", nil))
	assert.Equal(t, "resumo", mustFormat("resumo", assert.AnError))
}

func Test_parseIndiceEndereco(t *testing.T) {
	tests := []struct {
		name    string
		texto   string
		wantIdx int
		wantOK  bool
	}{
		{name: "número simples", texto: "1", wantIdx: 1, wantOK: true},
		{name: "número com parêntese", texto: "2)", wantIdx: 2, wantOK: true},
		{name: "número com ponto", texto: "3.", wantIdx: 3, wantOK: true},
		{name: "sem número", texto: "abc", wantIdx: 0, wantOK: false},
		{name: "vazio", texto: "", wantIdx: 0, wantOK: false},
		{name: "número extraído do texto", texto: "opção 3", wantIdx: 3, wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, ok := parseIndiceEndereco(tt.texto)
			assert.Equal(t, tt.wantIdx, idx)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func Test_pareceEndereco(t *testing.T) {
	tests := []struct {
		name  string
		texto string
		want  bool
	}{
		{name: "texto curto", texto: "Rua", want: false},
		{name: "sem número", texto: "Rua das Flores", want: false},
		{name: "rua com número e vírgula", texto: "Rua das Flores, 123", want: true},
		{name: "avenida abreviada", texto: "Av. Brasil 1000", want: true},
		{name: "só número sem indicador", texto: "12345", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pareceEndereco(tt.texto))
		})
	}
}

func Test_parseEnderecoTexto(t *testing.T) {
	t.Run("endereço completo com CEP", func(t *testing.T) {
		req := parseEnderecoTexto("Rua das Flores, 123, Centro, Pinhalzinho - SC, 89870-000")
		require.NotNil(t, req)
		assert.Equal(t, "Rua das Flores", req.Logradouro)
		assert.Equal(t, "123", req.Numero)
		assert.Equal(t, "Centro", req.Bairro)
		assert.Equal(t, "Pinhalzinho", req.Cidade)
		assert.Equal(t, "SC", req.Estado)
		assert.Equal(t, "89870-000", req.CEP)
		assert.Equal(t, "Brasil", req.Pais)
		assert.Equal(t, "entrega", req.Tipo)
	})

	t.Run("rua e número apenas", func(t *testing.T) {
		req := parseEnderecoTexto("Rua das Flores, 123")
		require.NotNil(t, req)
		assert.Equal(t, "Rua das Flores", req.Logradouro)
		assert.Equal(t, "123", req.Numero)
		assert.Equal(t, "", req.Bairro)
	})

	t.Run("sem número explícito extrai o número do texto", func(t *testing.T) {
		req := parseEnderecoTexto("Rua das Flores 123")
		require.NotNil(t, req)
		assert.Equal(t, "123", req.Numero)
	})

	t.Run("sem nenhum número usa S/N", func(t *testing.T) {
		req := parseEnderecoTexto("Rua das Flores")
		require.NotNil(t, req)
		assert.Equal(t, "S/N", req.Numero)
	})
}

func TestCarrinhoService_CalcularTotal(t *testing.T) {
	tests := []struct {
		name     string
		itens    []dto.ItemCarrinho
		expected float64
	}{
		{name: "carrinho vazio", itens: []dto.ItemCarrinho{}, expected: 0.0},
		{name: "um item", itens: []dto.ItemCarrinho{{Preco: 10.0, Quantidade: 2}}, expected: 20.0},
		{name: "vários itens", itens: []dto.ItemCarrinho{{Preco: 5.0, Quantidade: 3}, {Preco: 7.5, Quantidade: 2}}, expected: 15.0 + 15.0},
	}
	svc := &carrinhoService{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, svc.CalcularTotal(&dto.Carrinho{Itens: tt.itens}))
		})
	}
}

func TestCarrinhoService_CalcularTempoEstimado(t *testing.T) {
	svc := &carrinhoService{}

	t.Run("carrinho vazio", func(t *testing.T) {
		assert.Equal(t, 0, svc.CalcularTempoEstimado(&dto.Carrinho{}))
	})

	t.Run("um item", func(t *testing.T) {
		carrinho := &dto.Carrinho{Itens: []dto.ItemCarrinho{{Quantidade: 1}}}
		assert.Equal(t, 20, svc.CalcularTempoEstimado(carrinho)) // 15 + 1*5
	})

	t.Run("vários itens", func(t *testing.T) {
		carrinho := &dto.Carrinho{Itens: []dto.ItemCarrinho{{Quantidade: 2}, {Quantidade: 1}}}
		assert.Equal(t, 30, svc.CalcularTempoEstimado(carrinho)) // 15 + 3*5
	})
}

func TestCarrinhoService_mergeItem(t *testing.T) {
	svc := &carrinhoService{}

	t.Run("item novo é adicionado", func(t *testing.T) {
		carrinho := &dto.Carrinho{}
		item := dto.ItemCarrinho{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 1, Preco: 25.0}
		resultado := svc.mergeItem(carrinho, item)
		require.Len(t, resultado.Itens, 1)
		assert.Equal(t, "X-Bacon", resultado.Itens[0].ProdutoItem.Nome)
	})

	t.Run("item existente soma quantidade", func(t *testing.T) {
		carrinho := &dto.Carrinho{Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 1, Preco: 25.0}}}
		item := dto.ItemCarrinho{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 2, Preco: 25.0}
		resultado := svc.mergeItem(carrinho, item)
		require.Len(t, resultado.Itens, 1)
		assert.Equal(t, 3, resultado.Itens[0].Quantidade)
	})

	t.Run("observação do item sobrepõe a existente", func(t *testing.T) {
		carrinho := &dto.Carrinho{Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1}, Quantidade: 1, Observacao: "antiga"}}}
		item := dto.ItemCarrinho{ProdutoItem: dto.ProdutoItem{ID: 1}, Quantidade: 1, Observacao: "nova"}
		svc.mergeItem(carrinho, item)
		assert.Equal(t, "nova", carrinho.Itens[0].Observacao)
	})

	t.Run("item sem observação mantém a existente", func(t *testing.T) {
		carrinho := &dto.Carrinho{Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1}, Quantidade: 1, Observacao: "mantém"}}}
		item := dto.ItemCarrinho{ProdutoItem: dto.ProdutoItem{ID: 1}, Quantidade: 1}
		svc.mergeItem(carrinho, item)
		assert.Equal(t, "mantém", carrinho.Itens[0].Observacao)
	})
}

func TestCarrinhoService_FormatResumoCarrinho(t *testing.T) {
	svc := &carrinhoService{}

	t.Run("carrinho vazio", func(t *testing.T) {
		resumo, err := svc.FormatResumoCarrinho(testCtx(), &dto.Carrinho{})
		require.NoError(t, err)
		assert.Contains(t, resumo, "carrinho está vazio")
	})

	t.Run("com itens mostra total e tempo", func(t *testing.T) {
		carrinho := &dto.Carrinho{Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 2, Preco: 25.0}}}
		resumo, err := svc.FormatResumoCarrinho(testCtx(), carrinho)
		require.NoError(t, err)
		assert.Contains(t, resumo, "SEU CARRINHO")
		assert.Contains(t, resumo, "X-Bacon")
		assert.Contains(t, resumo, "R$ 50.00")
		assert.Contains(t, resumo, "25 minutos") // 15 + 2*5
	})
}

func TestCarrinhoService_FormatarPedidoConfirmado(t *testing.T) {
	svc := &carrinhoService{}

	t.Run("pedido com endereço", func(t *testing.T) {
		pedido := &dto.PedidoConfirmado{
			ID: 1, Total: 50.0, TempoEstimado: 25, ClienteNome: "João",
			EnderecoEntrega: &dto.EnderecoDTO{Logradouro: "Rua A", Numero: "123", Bairro: "Centro", Cidade: "Pinhalzinho", Estado: "SC"},
		}
		msg := svc.FormatarPedidoConfirmado(pedido)
		assert.Contains(t, msg, "PEDIDO CONFIRMADO")
		assert.Contains(t, msg, "Entrega em:** Rua A, 123")
	})

	t.Run("pedido sem endereço", func(t *testing.T) {
		pedido := &dto.PedidoConfirmado{ID: 2, Total: 50.0, TempoEstimado: 25}
		msg := svc.FormatarPedidoConfirmado(pedido)
		assert.Contains(t, msg, "PEDIDO CONFIRMADO")
		assert.NotContains(t, msg, "Entrega em")
	})
}

func TestCarrinhoService_GetCarrinho(t *testing.T) {
	t.Run("cache miss cria carrinho novo com estado aberto", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)

		carrinho, err := svc.GetCarrinho(testCtx(), 5, 1)
		require.NoError(t, err)
		require.NotNil(t, carrinho)
		assert.Equal(t, "5", carrinho.ClienteID)
		assert.Equal(t, "1", carrinho.TenantID)
		assert.Empty(t, carrinho.Itens)
		assert.Equal(t, dto.EstadoAberto, carrinho.Estado)
	})

	t.Run("cache hit retorna carrinho existente", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		existente := dto.Carrinho{
			ClienteID: "5", TenantID: "1",
			Itens:  []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 2, Preco: 25.0}},
			Estado: dto.EstadoAberto,
		}
		mockRedisCacheHit(t, redisMock, existente)

		carrinho, err := svc.GetCarrinho(testCtx(), 5, 1)
		require.NoError(t, err)
		require.Len(t, carrinho.Itens, 1)
		assert.Equal(t, "X-Bacon", carrinho.Itens[0].ProdutoItem.Nome)
	})

	t.Run("estado vazio vira aberto", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: []dto.ItemCarrinho{}})

		carrinho, err := svc.GetCarrinho(testCtx(), 5, 1)
		require.NoError(t, err)
		assert.Equal(t, dto.EstadoAberto, carrinho.Estado)
	})
}

func TestCarrinhoService_AdicionarItem(t *testing.T) {
	t.Run("cache miss: cria carrinho e salva com o item", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)

		var salvo *dto.Carrinho
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), "carrinho:1:5", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, value any, ttl time.Duration) error {
				salvo = value.(*dto.Carrinho)
				return nil
			})

		item := dto.ItemCarrinho{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 1, Preco: 25.0}
		err := svc.AdicionarItem(testCtx(), 5, 1, item)
		require.NoError(t, err)
		require.NotNil(t, salvo)
		require.Len(t, salvo.Itens, 1)
		assert.Equal(t, "X-Bacon", salvo.Itens[0].ProdutoItem.Nome)
		assert.Equal(t, dto.EstadoAberto, salvo.Estado)
	})

	t.Run("cache hit: mescla com item existente", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{
			ClienteID: "5", TenantID: "1",
			Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 1, Preco: 25.0}},
		})
		var salvo *dto.Carrinho
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, value any, ttl time.Duration) error {
				salvo = value.(*dto.Carrinho)
				return nil
			})

		item := dto.ItemCarrinho{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 2, Preco: 25.0}
		err := svc.AdicionarItem(testCtx(), 5, 1, item)
		require.NoError(t, err)
		require.NotNil(t, salvo)
		require.Len(t, salvo.Itens, 1)
		assert.Equal(t, 3, salvo.Itens[0].Quantidade)
	})
}

func TestCarrinhoService_RemoverItem(t *testing.T) {
	t.Run("quantidade zero remove o item por completo", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{
			ClienteID: "5", TenantID: "1",
			Itens: []dto.ItemCarrinho{
				{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 2, Preco: 25.0},
				{ProdutoItem: dto.ProdutoItem{ID: 2, Nome: "Coca"}, Quantidade: 1, Preco: 8.0},
			},
		})
		var salvo *dto.Carrinho
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, value any, ttl time.Duration) error {
				salvo = value.(*dto.Carrinho)
				return nil
			})

		err := svc.RemoverItem(testCtx(), 5, 1, dto.ItemCarrinho{ProdutoItem: dto.ProdutoItem{ID: 1}}, 0)
		require.NoError(t, err)
		require.NotNil(t, salvo)
		require.Len(t, salvo.Itens, 1)
		assert.Equal(t, "Coca", salvo.Itens[0].ProdutoItem.Nome)
	})

	t.Run("quantidade parcial decrementa", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{
			ClienteID: "5", TenantID: "1",
			Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 3, Preco: 25.0}},
		})
		var salvo *dto.Carrinho
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, value any, ttl time.Duration) error {
				salvo = value.(*dto.Carrinho)
				return nil
			})

		err := svc.RemoverItem(testCtx(), 5, 1, dto.ItemCarrinho{ProdutoItem: dto.ProdutoItem{ID: 1}}, 1)
		require.NoError(t, err)
		require.Len(t, salvo.Itens, 1)
		assert.Equal(t, 2, salvo.Itens[0].Quantidade)
	})

	t.Run("item não encontrado retorna erro", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{
			ClienteID: "5", TenantID: "1",
			Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 2, Preco: 25.0}},
		})

		err := svc.RemoverItem(testCtx(), 5, 1, dto.ItemCarrinho{ProdutoItem: dto.ProdutoItem{ID: 99}}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "não encontrado")
	})
}

func TestCarrinhoService_LimparCarrinho(t *testing.T) {
	svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
	redisMock.EXPECT().DeleteWithContext(testCtx(), "carrinho:1:5").Return(nil)
	require.NoError(t, svc.LimparCarrinho(testCtx(), 5, 1))
}

func TestCarrinhoService_FormatResumoCarrinhoByCliente(t *testing.T) {
	svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
	mockRedisCacheHit(t, redisMock, dto.Carrinho{
		ClienteID: "5", TenantID: "1",
		Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 2, Preco: 25.0}},
	})

	resumo, err := svc.FormatResumoCarrinhoByCliente(testCtx(), 5, 1)
	require.NoError(t, err)
	assert.Contains(t, resumo, "X-Bacon")
}

func TestCarrinhoService_FinalizarCarrinhoComEndereco(t *testing.T) {
	t.Run("carrinho vazio retorna erro", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)

		pedido, err := svc.FinalizarCarrinhoComEndereco(testCtx(), 5, 1, "João", 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "carrinho vazio")
		assert.Nil(t, pedido)
	})

	t.Run("sucesso monta pedido extraído e limpa carrinho", func(t *testing.T) {
		svc, redisMock, _, pedidoMock, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{
			ClienteID: "5", TenantID: "1",
			Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 2, Preco: 25.0, Observacao: "sem cebola"}},
		})
		redisMock.EXPECT().DeleteWithContext(testCtx(), "carrinho:1:5").Return(nil)

		pedidoMock.EXPECT().ProcessarPedidoComEndereco(testCtx(), uint(1), uint(5), "João", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, tenantID, clienteID uint, nome string, pe *dto.PedidoExtraido, end *uint) (*dto.PedidoConfirmado, error) {
				require.Len(t, pe.Itens, 1)
				assert.Equal(t, "X-Bacon", pe.Itens[0].ProdutoItem.Nome)
				assert.Equal(t, 2, pe.Itens[0].Quantidade)
				assert.Equal(t, "sem cebola", pe.Itens[0].Observacao)
				assert.Equal(t, 25.0, pe.Itens[0].PrecoUnitario)
				assert.Nil(t, end)
				return &dto.PedidoConfirmado{ID: 99, Total: 50.0, TempoEstimado: 25}, nil
			})

		pedido, err := svc.FinalizarCarrinhoComEndereco(testCtx(), 5, 1, "João", 0)
		require.NoError(t, err)
		require.NotNil(t, pedido)
		assert.Equal(t, 99, pedido.ID)
	})

	t.Run("enderecoID diferente de zero passa ponteiro", func(t *testing.T) {
		svc, redisMock, _, pedidoMock, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{
			ClienteID: "5", TenantID: "1",
			Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 1, Preco: 25.0}},
		})
		redisMock.EXPECT().DeleteWithContext(testCtx(), "carrinho:1:5").Return(nil)

		pedidoMock.EXPECT().ProcessarPedidoComEndereco(testCtx(), uint(1), uint(5), "João", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, tenantID, clienteID uint, nome string, pe *dto.PedidoExtraido, end *uint) (*dto.PedidoConfirmado, error) {
				require.NotNil(t, end)
				assert.Equal(t, uint(10), *end)
				return &dto.PedidoConfirmado{ID: 100, Total: 25.0}, nil
			})

		pedido, err := svc.FinalizarCarrinhoComEndereco(testCtx(), 5, 1, "João", 10)
		require.NoError(t, err)
		assert.Equal(t, 100, pedido.ID)
	})
}

func TestCarrinhoService_FinalizarCarrinho(t *testing.T) {
	svc, redisMock, _, pedidoMock, _, _, _ := novoCarrinhoServiceMock(t)
	mockRedisCacheHit(t, redisMock, dto.Carrinho{
		ClienteID: "5", TenantID: "1",
		Itens: []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 1, Preco: 25.0}},
	})
	redisMock.EXPECT().DeleteWithContext(testCtx(), "carrinho:1:5").Return(nil)
	pedidoMock.EXPECT().ProcessarPedidoComEndereco(testCtx(), uint(1), uint(5), "João", gomock.Any(), gomock.Any()).
		Return(&dto.PedidoConfirmado{ID: 7}, nil)

	pedido, err := svc.FinalizarCarrinho(testCtx(), 5, 1, "João")
	require.NoError(t, err)
	assert.Equal(t, 7, pedido.ID)
}

func carrinhoComItem() []dto.ItemCarrinho {
	return []dto.ItemCarrinho{{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 1, Preco: 25.0}}
}

func TestCarrinhoService_iniciarFluxoFinalizacao(t *testing.T) {
	t.Run("carrinho vazio", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)

		msg, err := svc.iniciarFluxoFinalizacao(testCtx(), 5, 1)
		require.NoError(t, err)
		assert.Contains(t, msg, "carrinho está vazio")
	})

	t.Run("sem endereços: aguarda novo endereço", func(t *testing.T) {
		svc, redisMock, _, _, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: carrinhoComItem()})
		var salvo *dto.Carrinho
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, value any, ttl time.Duration) error {
				salvo = value.(*dto.Carrinho)
				return nil
			})
		clienteMock.EXPECT().ListarEnderecos(testCtx(), uint(5)).Return([]dto.EnderecoDTO{}, nil)

		msg, err := svc.iniciarFluxoFinalizacao(testCtx(), 5, 1)
		require.NoError(t, err)
		assert.Contains(t, msg, "ENDEREÇO DE ENTREGA")
		require.NotNil(t, salvo)
		assert.Equal(t, dto.EstadoAguardandoEnderecoNovo, salvo.Estado)
	})

	t.Run("com endereços: confirma o mais recente", func(t *testing.T) {
		svc, redisMock, _, _, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: carrinhoComItem()})
		var salvo *dto.Carrinho
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, value any, ttl time.Duration) error {
				salvo = value.(*dto.Carrinho)
				return nil
			})
		clienteMock.EXPECT().ListarEnderecos(testCtx(), uint(5)).
			Return([]dto.EnderecoDTO{{ID: 10, Logradouro: "Rua A", Numero: "123"}}, nil)

		msg, err := svc.iniciarFluxoFinalizacao(testCtx(), 5, 1)
		require.NoError(t, err)
		assert.Contains(t, msg, "Confirma entrega em")
		require.NotNil(t, salvo)
		assert.Equal(t, dto.EstadoAguardandoConfirmacaoEndereco, salvo.Estado)
		assert.Equal(t, 0, salvo.EnderecoConfirmacaoIdx)
	})
}

func TestCarrinhoService_handleSelecaoEndereco(t *testing.T) {
	itens := carrinhoComItem()

	t.Run("novo endereço", func(t *testing.T) {
		svc, redisMock, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Estado: dto.EstadoAguardandoEnderecoLista, Itens: itens}

		msg, err := svc.handleSelecaoEndereco(testCtx(), 5, 1, carrinho, "novo")
		require.NoError(t, err)
		assert.Contains(t, msg, "NOVO ENDEREÇO")
		assert.Equal(t, dto.EstadoAguardandoEnderecoNovo, carrinho.Estado)
	})

	t.Run("índice válido finaliza pedido", func(t *testing.T) {
		svc, redisMock, _, pedidoMock, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		enderecos := []dto.EnderecoDTO{{ID: 10, Logradouro: "Rua A", Numero: "123"}}
		clienteMock.EXPECT().ListarEnderecos(testCtx(), uint(5)).Return(enderecos, nil)
		clienteMock.EXPECT().FindByID(testCtx(), uint(5)).Return(&dto.ClienteDTO{ID: 5, Nome: "João"}, nil)
		clienteMock.EXPECT().AtualizarUltimoPedido(testCtx(), uint(5)).Return(nil)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: itens})
		redisMock.EXPECT().DeleteWithContext(testCtx(), "carrinho:1:5").Return(nil)
		pedidoMock.EXPECT().ProcessarPedidoComEndereco(testCtx(), uint(1), uint(5), "João", gomock.Any(), gomock.Any()).
			Return(&dto.PedidoConfirmado{ID: 1, Total: 25.0, TempoEstimado: 20}, nil)

		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Estado: dto.EstadoAguardandoEnderecoLista, Itens: itens}
		msg, err := svc.handleSelecaoEndereco(testCtx(), 5, 1, carrinho, "1")
		require.NoError(t, err)
		assert.Contains(t, msg, "PEDIDO CONFIRMADO")
	})

	t.Run("índice inválido incrementa tentativas", func(t *testing.T) {
		svc, redisMock, _, _, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		enderecos := []dto.EnderecoDTO{{ID: 10, Logradouro: "Rua A", Numero: "123"}}
		clienteMock.EXPECT().ListarEnderecos(testCtx(), uint(5)).Return(enderecos, nil)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Estado: dto.EstadoAguardandoEnderecoLista, Itens: itens}

		msg, err := svc.handleSelecaoEndereco(testCtx(), 5, 1, carrinho, "9")
		require.NoError(t, err)
		assert.Contains(t, msg, "Opção inválida")
		assert.Equal(t, 1, carrinho.TentativasEndereco)
	})

	t.Run("tentativa limite mostra lista", func(t *testing.T) {
		svc, redisMock, _, _, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		enderecos := []dto.EnderecoDTO{{ID: 10, Logradouro: "Rua A", Numero: "123"}}
		clienteMock.EXPECT().ListarEnderecos(testCtx(), uint(5)).Return(enderecos, nil)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Estado: dto.EstadoAguardandoEnderecoLista, Itens: itens, TentativasEndereco: 1}

		msg, err := svc.handleSelecaoEndereco(testCtx(), 5, 1, carrinho, "9")
		require.NoError(t, err)
		assert.Contains(t, msg, "SEUS ENDEREÇOS CADASTRADOS")
		assert.Equal(t, 2, carrinho.TentativasEndereco)
	})
}

func TestCarrinhoService_handleNovoEndereco(t *testing.T) {
	itens := carrinhoComItem()

	t.Run("endereço curto", func(t *testing.T) {
		svc, _, _, _, _, _, _ := novoCarrinhoServiceMock(t)
		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Estado: dto.EstadoAguardandoEnderecoNovo, Itens: itens}

		msg, err := svc.handleNovoEndereco(testCtx(), 5, 1, carrinho, "Rua A")
		require.NoError(t, err)
		assert.Contains(t, msg, "curto")
	})

	t.Run("endereço válido cadastra e finaliza", func(t *testing.T) {
		svc, redisMock, _, pedidoMock, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		novoEndereco := &dto.EnderecoDTO{ID: 20, Logradouro: "Rua das Flores", Numero: "123", Bairro: "Centro", Cidade: "Pinhalzinho", Estado: "SC"}
		clienteMock.EXPECT().AdicionarEndereco(testCtx(), uint(5), gomock.Any()).Return(novoEndereco, nil)
		clienteMock.EXPECT().FindByID(testCtx(), uint(5)).Return(&dto.ClienteDTO{ID: 5, Nome: "João"}, nil)
		clienteMock.EXPECT().AtualizarUltimoPedido(testCtx(), uint(5)).Return(nil)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockRedisCacheHit(t, redisMock, dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: itens})
		redisMock.EXPECT().DeleteWithContext(testCtx(), "carrinho:1:5").Return(nil)
		pedidoMock.EXPECT().ProcessarPedidoComEndereco(testCtx(), uint(1), uint(5), "João", gomock.Any(), gomock.Any()).
			Return(&dto.PedidoConfirmado{ID: 2, Total: 25.0}, nil)

		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Estado: dto.EstadoAguardandoEnderecoNovo, Itens: itens}
		msg, err := svc.handleNovoEndereco(testCtx(), 5, 1, carrinho, "Rua das Flores, 123, Centro, Pinhalzinho - SC, 89870-000")
		require.NoError(t, err)
		assert.Contains(t, msg, "Endereço cadastrado")
		assert.Contains(t, msg, "PEDIDO CONFIRMADO")
	})

	t.Run("falha ao salvar endereço", func(t *testing.T) {
		svc, _, _, _, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		clienteMock.EXPECT().AdicionarEndereco(testCtx(), uint(5), gomock.Any()).Return(nil, assert.AnError)
		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Estado: dto.EstadoAguardandoEnderecoNovo, Itens: itens}

		msg, err := svc.handleNovoEndereco(testCtx(), 5, 1, carrinho, "Rua das Flores, 123, Centro")
		require.NoError(t, err)
		assert.Contains(t, msg, "Erro ao salvar endereço")
	})
}

func TestCarrinhoService_finalizarComEndereco(t *testing.T) {
	itens := carrinhoComItem()

	t.Run("sucesso", func(t *testing.T) {
		svc, redisMock, _, pedidoMock, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		clienteMock.EXPECT().FindByID(testCtx(), uint(5)).Return(&dto.ClienteDTO{ID: 5, Nome: "João"}, nil)
		clienteMock.EXPECT().AtualizarUltimoPedido(testCtx(), uint(5)).Return(nil)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: itens})
		redisMock.EXPECT().DeleteWithContext(testCtx(), "carrinho:1:5").Return(nil)
		pedidoMock.EXPECT().ProcessarPedidoComEndereco(testCtx(), uint(1), uint(5), "João", gomock.Any(), gomock.Any()).
			Return(&dto.PedidoConfirmado{ID: 3, Total: 25.0}, nil)

		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: itens}
		msg, err := svc.finalizarComEndereco(testCtx(), 5, 1, carrinho, 10)
		require.NoError(t, err)
		assert.Contains(t, msg, "PEDIDO CONFIRMADO")
		assert.Equal(t, dto.EstadoAberto, carrinho.Estado)
		require.NotNil(t, carrinho.EnderecoID)
		assert.Equal(t, uint(10), *carrinho.EnderecoID)
	})

	t.Run("carrinho vazio", func(t *testing.T) {
		svc, redisMock, _, _, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		clienteMock.EXPECT().FindByID(testCtx(), uint(5)).Return(&dto.ClienteDTO{ID: 5, Nome: "João"}, nil)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockRedisCacheMiss(redisMock) // FinalizarCarrinhoComEndereco encontra carrinho vazio

		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: itens}
		msg, err := svc.finalizarComEndereco(testCtx(), 5, 1, carrinho, 10)
		require.NoError(t, err)
		assert.Contains(t, msg, "carrinho está vazio")
	})

	t.Run("erro genérico ao finalizar", func(t *testing.T) {
		svc, redisMock, _, pedidoMock, clienteMock, _, _ := novoCarrinhoServiceMock(t)
		clienteMock.EXPECT().FindByID(testCtx(), uint(5)).Return(&dto.ClienteDTO{ID: 5, Nome: "João"}, nil)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockRedisCacheHit(t, redisMock, dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: itens})
		pedidoMock.EXPECT().ProcessarPedidoComEndereco(testCtx(), uint(1), uint(5), "João", gomock.Any(), gomock.Any()).
			Return(nil, assert.AnError)

		carrinho := &dto.Carrinho{ClienteID: "5", TenantID: "1", Itens: itens}
		msg, err := svc.finalizarComEndereco(testCtx(), 5, 1, carrinho, 10)
		require.NoError(t, err)
		assert.Contains(t, msg, "Erro ao finalizar pedido")
	})
}

// mockObterTextoBaseReturnInput simula o ObterTextoBase devolvendo o texto puro do input.
func mockObterTextoBaseReturnInput(llmMock *mocks.MockLLMServiceInterface) {
	llmMock.EXPECT().ObterTextoBase(testCtx(), uint(1), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tenantID uint, input dto.MessageInput) (string, error) {
			return input.Text, nil
		}).AnyTimes()
}

func TestCarrinhoService_ProcessarMensagem(t *testing.T) {
	t.Run("ObterTextoBase com erro retorna fallback", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		llmMock.EXPECT().ObterTextoBase(testCtx(), uint(1), gomock.Any()).Return("", assert.AnError)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "Não consegui entender")
	})

	t.Run("texto vazio", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "   ", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "Não entendi")
	})

	t.Run("saudação no fast-path", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "oi", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "O que vai querer hoje?")
	})

	t.Run("agradecimento no fast-path", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "obrigado", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "Por nada!")
	})

	t.Run("smalltalk no fast-path", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "tudo bem", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "Tudo ótimo por aqui!")
	})

	t.Run("ver carrinho no fast-path", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "ver carrinho", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "carrinho está vazio")
	})

	t.Run("ClassificarEExtrairKeywords com erro", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)
		llmMock.EXPECT().ClassificarEExtrairKeywords(testCtx(), uint(1), gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "quero um x-bacon", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "Tive um problema técnico")
	})
}

func TestCarrinhoService_ProcessarMensagem_Acoes(t *testing.T) {
	t.Run("ação conversa retorna resposta", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)
		llmMock.EXPECT().ClassificarEExtrairKeywords(testCtx(), uint(1), gomock.Any(), gomock.Any()).
			Return(&llm.IntencaoEKeywordsResult{Acao: "conversa", Resposta: "Funcionamos das 18h às 23h"}, nil)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "qual o horário?", Source: models.SourceText})
		require.NoError(t, err)
		assert.Equal(t, "Funcionamos das 18h às 23h", msg)
	})

	t.Run("ação listar_categorias", func(t *testing.T) {
		svc, redisMock, cardapioMock, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)
		llmMock.EXPECT().ClassificarEExtrairKeywords(testCtx(), uint(1), gomock.Any(), gomock.Any()).
			Return(&llm.IntencaoEKeywordsResult{Acao: "listar_categorias"}, nil)
		cardapioMock.EXPECT().GetCardapio(testCtx(), uint(1)).
			Return([]dto.ProdutoItem{{ID: 1, Nome: "X-Bacon", Categoria: "Lanches"}}, nil)
		cardapioMock.EXPECT().ListarCategoriasHumanizado(gomock.Any()).Return("Temos Lanches, Bebidas.")

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "o que tem?", Source: models.SourceText})
		require.NoError(t, err)
		assert.Equal(t, "Temos Lanches, Bebidas.", msg)
	})

	t.Run("limpar carrinho no fast-path", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)
		redisMock.EXPECT().DeleteWithContext(testCtx(), "carrinho:1:5").Return(nil)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "limpar carrinho", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "Carrinho limpo!")
	})

	t.Run("ação limpar_carrinho via LLM", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)
		llmMock.EXPECT().ClassificarEExtrairKeywords(testCtx(), uint(1), gomock.Any(), gomock.Any()).
			Return(&llm.IntencaoEKeywordsResult{Acao: "limpar_carrinho"}, nil)
		redisMock.EXPECT().DeleteWithContext(testCtx(), "carrinho:1:5").Return(nil)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "quero zerar", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "Carrinho limpo!")
	})

	t.Run("ação finalizar com carrinho vazio", func(t *testing.T) {
		svc, redisMock, _, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)
		llmMock.EXPECT().ClassificarEExtrairKeywords(testCtx(), uint(1), gomock.Any(), gomock.Any()).
			Return(&llm.IntencaoEKeywordsResult{Acao: "finalizar"}, nil)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "finalizar pedido", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "carrinho está vazio")
	})

	t.Run("adicionar item no cardápio pequeno", func(t *testing.T) {
		svc, redisMock, cardapioMock, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)
		llmMock.EXPECT().ClassificarEExtrairKeywords(testCtx(), uint(1), gomock.Any(), gomock.Any()).
			Return(&llm.IntencaoEKeywordsResult{Acao: "adicionar", Keywords: []llm.LLMKeywordItemResult{{Nome: "x-bacon"}}}, nil)
		cardapioMock.EXPECT().GetCardapio(testCtx(), uint(1)).
			Return([]dto.ProdutoItem{{ID: 1, Nome: "X-Bacon", Preco: 25.0}}, nil)
		llmMock.EXPECT().ResolveItemsByMenu(testCtx(), uint(1), gomock.Any(), gomock.Any()).
			Return(&dto.IntencaoCliente{Acao: "adicionar", Itens: []dto.ItemCarrinho{
				{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 1, Preco: 25.0},
			}}, nil)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "quero um x-bacon", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "SEU CARRINHO")
		assert.Contains(t, msg, "X-Bacon")
	})

	t.Run("adicionar item no cardápio grande", func(t *testing.T) {
		svc, redisMock, cardapioMock, _, _, _, llmMock := novoCarrinhoServiceMock(t)
		mockRedisCacheMiss(redisMock)
		mockObterTextoBaseReturnInput(llmMock)
		llmMock.EXPECT().ClassificarEExtrairKeywords(testCtx(), uint(1), gomock.Any(), gomock.Any()).
			Return(&llm.IntencaoEKeywordsResult{Acao: "adicionar", Keywords: []llm.LLMKeywordItemResult{{Nome: "x-bacon"}}}, nil)

		var cardapioGrande []dto.ProdutoItem
		for i := 1; i <= limiteCardapioPequeno+1; i++ {
			cardapioGrande = append(cardapioGrande, dto.ProdutoItem{ID: uint(i), Preco: 10.0})
		}
		cardapioMock.EXPECT().GetCardapio(testCtx(), uint(1)).Return(cardapioGrande, nil)
		cardapioMock.EXPECT().ReduzirPorKeywords(testCtx(), uint(1), gomock.Any()).
			Return([]dto.ProdutoItem{{ID: 1, Nome: "X-Bacon", Preco: 25.0}}, nil)
		llmMock.EXPECT().ResolverItensByKeyWords(testCtx(), uint(1), gomock.Any(), gomock.Any()).
			Return([]dto.ItemCarrinho{
				{ProdutoItem: dto.ProdutoItem{ID: 1, Nome: "X-Bacon"}, Quantidade: 1, Preco: 25.0},
			}, nil)
		redisMock.EXPECT().SetJSONWithContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		msg, err := svc.ProcessarMensagem(testCtx(), 5, 1, dto.MessageInput{Text: "quero um x-bacon", Source: models.SourceText})
		require.NoError(t, err)
		assert.Contains(t, msg, "X-Bacon")
	})
}
