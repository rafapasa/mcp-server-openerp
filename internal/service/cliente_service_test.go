// internal/service/cliente_service_test.go
package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/mocks"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// novoClienteServiceMock cria o serviço com repositórios mockados via gomock.
func novoClienteServiceMock(t *testing.T) (*clienteService, *mocks.MockClienteRepositoryInterface, *mocks.MockEnderecoRepositoryInterface) {
	t.Helper()

	ctrl := gomock.NewController(t)
	clienteRepo := mocks.NewMockClienteRepositoryInterface(ctrl)
	enderecoRepo := mocks.NewMockEnderecoRepositoryInterface(ctrl)

	svc := &clienteService{
		clienteRepo:  clienteRepo,
		enderecoRepo: enderecoRepo,
	}
	return svc, clienteRepo, enderecoRepo
}

// testCtx retorna um contexto com logger no-op (evita panic do logger global não inicializado).
func testCtx() context.Context {
	return logger.WithLogger(context.Background(), zap.NewNop())
}

func TestClienteService_Create(t *testing.T) {
	telefoneValido := "5547999999999"
	tenantID := uint(1)

	t.Run("erro: tenant_id obrigatório", func(t *testing.T) {
		svc, _, _ := novoClienteServiceMock(t)
		resultado, err := svc.Create(testCtx(), &dto.CriarClienteRequest{Telefone: telefoneValido, Nome: "João"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_id")
		assert.Nil(t, resultado)
	})

	t.Run("erro: telefone vazio", func(t *testing.T) {
		svc, _, _ := novoClienteServiceMock(t)
		resultado, err := svc.Create(testCtx(), &dto.CriarClienteRequest{TenantID: tenantID, Nome: "João"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "telefone")
		assert.Nil(t, resultado)
	})

	t.Run("erro: telefone inválido", func(t *testing.T) {
		svc, _, _ := novoClienteServiceMock(t)
		resultado, err := svc.Create(testCtx(), &dto.CriarClienteRequest{TenantID: tenantID, Telefone: "abc", Nome: "João"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "telefone")
		assert.Nil(t, resultado)
	})

	t.Run("FindByTelefone com erro não-notfound: propaga e não cria", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByTelefone(testCtx(), telefoneValido, tenantID).Return(nil, errors.New("banco fora"))

		resultado, err := svc.Create(testCtx(), &dto.CriarClienteRequest{TenantID: tenantID, Telefone: telefoneValido, Nome: "João"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "banco fora")
		assert.Nil(t, resultado)
	})

	t.Run("cliente já existe ativo: retorna existente sem criar", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		existente := &models.Cliente{ID: 7, TenantID: 1, Telefone: "5547999999999", Nome: "João", Status: models.StatusClienteAtivo}
		clienteRepo.EXPECT().FindByTelefone(testCtx(), "5547999999999", uint(1)).Return(existente, nil)

		resultado, err := svc.Create(testCtx(), &dto.CriarClienteRequest{TenantID: 1, Telefone: "5547999999999", Nome: "João"})
		require.NoError(t, err)
		assert.Equal(t, uint(7), resultado.ID)
		assert.Equal(t, models.StatusClienteAtivo, resultado.Status)
	})

	t.Run("cliente existe inativo: reativa e retorna atualizado", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		inativo := &models.Cliente{ID: 7, TenantID: 1, Telefone: "5547999999999", Nome: "João", Status: models.StatusClienteInativo}
		reativado := &models.Cliente{ID: 7, TenantID: 1, Telefone: "5547999999999", Nome: "João", Status: models.StatusClienteAtivo}

		// 1ª busca (dentro de Create) retorna inativo
		clienteRepo.EXPECT().FindByTelefone(testCtx(), "5547999999999", uint(1)).Return(inativo, nil)
		// ReativarCliente -> AtualizarStatus -> FindByID + Update
		clienteRepo.EXPECT().FindByID(testCtx(), uint(7)).Return(inativo, nil)
		clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).Return(nil)
		// 2ª busca após reativar retorna o cliente atualizado
		clienteRepo.EXPECT().FindByTelefone(testCtx(), "5547999999999", uint(1)).Return(reativado, nil)

		resultado, err := svc.Create(testCtx(), &dto.CriarClienteRequest{TenantID: 1, Telefone: "5547999999999", Nome: "João"})
		require.NoError(t, err)
		assert.Equal(t, uint(7), resultado.ID)
		assert.Equal(t, models.StatusClienteAtivo, resultado.Status)
	})

	t.Run("cliente novo: cria com NomePerfil = Nome e status ativo", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByTelefone(testCtx(), "5547999999999", uint(1)).Return(nil, nil)
		clienteRepo.EXPECT().Create(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.Equal(t, uint(1), c.TenantID)
			assert.Equal(t, "5547999999999", c.Telefone)
			assert.Equal(t, "João", c.Nome)
			assert.Equal(t, "João", c.NomePerfil)
			assert.Equal(t, models.StatusClienteAtivo, c.Status)
			c.ID = 42
			return nil
		})

		resultado, err := svc.Create(testCtx(), &dto.CriarClienteRequest{TenantID: 1, Telefone: "5547999999999", Nome: "João"})
		require.NoError(t, err)
		assert.Equal(t, uint(42), resultado.ID)
	})

	t.Run("nome vazio: usa NomePerfil como Nome", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByTelefone(testCtx(), "5547999999999", uint(1)).Return(nil, nil)
		clienteRepo.EXPECT().Create(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.Equal(t, "Perfil WhatsApp", c.Nome)
			assert.Equal(t, "Perfil WhatsApp", c.NomePerfil)
			return nil
		})

		_, err := svc.Create(testCtx(), &dto.CriarClienteRequest{TenantID: 1, Telefone: "5547999999999", Nome: "", NomePerfil: "Perfil WhatsApp"})
		require.NoError(t, err)
	})

	t.Run("erro: repo.Create falha", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByTelefone(testCtx(), "5547999999999", uint(1)).Return(nil, nil)
		clienteRepo.EXPECT().Create(testCtx(), gomock.Any()).Return(errors.New("erro no banco"))

		resultado, err := svc.Create(testCtx(), &dto.CriarClienteRequest{TenantID: 1, Telefone: "5547999999999", Nome: "João"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "erro ao criar cliente")
		assert.Nil(t, resultado)
	})
}

func TestClienteService_BuscarOuCriarPorTelefone(t *testing.T) {
	telefone := "5547999999999"
	tenantID := uint(1)

	t.Run("não existe: cria via Create", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		// 1ª busca: dentro do BuscarOuCriarPorTelefone
		clienteRepo.EXPECT().FindByTelefone(testCtx(), telefone, tenantID).Return(nil, nil)
		// 2ª busca: nova consulta dentro do Create delegado
		clienteRepo.EXPECT().FindByTelefone(testCtx(), telefone, tenantID).Return(nil, nil)
		clienteRepo.EXPECT().Create(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.Equal(t, "Perfil", c.Nome)
			assert.Equal(t, "Perfil", c.NomePerfil)
			c.ID = 1
			return nil
		})

		resultado, err := svc.BuscarOuCriarPorTelefone(testCtx(), tenantID, telefone, "Perfil")
		require.NoError(t, err)
		assert.Equal(t, uint(1), resultado.ID)
	})

	t.Run("existe ativo com mesmo nome: retorna sem atualizar", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		existente := &models.Cliente{ID: 3, TenantID: tenantID, Telefone: telefone, Nome: "João", NomePerfil: "João", Status: models.StatusClienteAtivo}
		clienteRepo.EXPECT().FindByTelefone(testCtx(), telefone, tenantID).Return(existente, nil)

		resultado, err := svc.BuscarOuCriarPorTelefone(testCtx(), tenantID, telefone, "João")
		require.NoError(t, err)
		assert.Equal(t, uint(3), resultado.ID)
	})

	t.Run("existe ativo com nome diferente: atualiza NomePerfil", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		// "Maria" vs "João" -> Jaro-Winkler ~0.53 (< 0.80), entra no branch de atualização
		existente := &models.Cliente{ID: 3, TenantID: tenantID, Telefone: telefone, Nome: "Maria", NomePerfil: "Maria", Status: "ativo"}
		clienteRepo.EXPECT().FindByTelefone(testCtx(), telefone, tenantID).Return(existente, nil)
		clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.Equal(t, "João", c.NomePerfil)
			return nil
		})

		resultado, err := svc.BuscarOuCriarPorTelefone(testCtx(), tenantID, telefone, "João")
		require.NoError(t, err)
		assert.Equal(t, "João", resultado.NomePerfil)
	})

	t.Run("existe inativo: reativa", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		inativo := &models.Cliente{ID: 3, TenantID: tenantID, Telefone: telefone, Nome: "João", NomePerfil: "João", Status: models.StatusClienteInativo}
		clienteRepo.EXPECT().FindByTelefone(testCtx(), telefone, tenantID).Return(inativo, nil)
		// ReativarCliente -> AtualizarStatus -> FindByID + Update
		clienteRepo.EXPECT().FindByID(testCtx(), uint(3)).Return(inativo, nil)
		clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.Equal(t, models.StatusClienteAtivo, c.Status)
			return nil
		})

		resultado, err := svc.BuscarOuCriarPorTelefone(testCtx(), tenantID, telefone, "João")
		require.NoError(t, err)
		assert.Equal(t, models.StatusClienteAtivo, resultado.Status)
	})

	t.Run("erro: FindByTelefone retorna erro inesperado", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByTelefone(testCtx(), telefone, tenantID).Return(nil, errors.New("banco fora"))

		resultado, err := svc.BuscarOuCriarPorTelefone(testCtx(), tenantID, telefone, "Perfil")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "banco fora")
		assert.Nil(t, resultado)
	})
}

func TestClienteService_AdicionarEndereco(t *testing.T) {
	reqValido := &dto.CriarEnderecoRequest{
		Logradouro: "Rua Teste",
		Numero:     "123",
		Bairro:     "Centro",
		Cidade:     "Pinhalzinho",
		Estado:     "SC",
		CEP:        "89870-000",
		Tipo:       "entrega",
	}

	t.Run("erro: cliente não encontrado", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(nil, gorm.ErrRecordNotFound)

		resultado, err := svc.AdicionarEndereco(testCtx(), 1, reqValido)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cliente não encontrado")
		assert.Nil(t, resultado)
	})

	t.Run("erro: logradouro vazio", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(&models.Cliente{ID: 1}, nil)

		_, err := svc.AdicionarEndereco(testCtx(), 1, &dto.CriarEnderecoRequest{Numero: "123"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "logradouro")
	})

	t.Run("erro: número vazio", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(&models.Cliente{ID: 1}, nil)

		_, err := svc.AdicionarEndereco(testCtx(), 1, &dto.CriarEnderecoRequest{Logradouro: "Rua Teste"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "número")
	})

	t.Run("sucesso: endereço normal (principal=false) não chama Update", func(t *testing.T) {
		svc, clienteRepo, enderecoRepo := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(&models.Cliente{ID: 1}, nil)
		enderecoRepo.EXPECT().Create(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, e *models.Endereco) error {
			assert.Equal(t, uint(1), e.ClienteID)
			assert.Equal(t, "Rua Teste", e.Logradouro)
			assert.Equal(t, "123", e.Numero)
			assert.Equal(t, "entrega", e.Tipo)
			e.ID = 99
			return nil
		})

		resultado, err := svc.AdicionarEndereco(testCtx(), 1, reqValido)
		require.NoError(t, err)
		assert.Equal(t, uint(99), resultado.ID)
	})

	t.Run("sucesso: principal=true desmarca anteriores via UnsetPrincipalByCliente", func(t *testing.T) {
		svc, clienteRepo, enderecoRepo := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(&models.Cliente{ID: 1}, nil)
		req := *reqValido
		req.Principal = true

		enderecoRepo.EXPECT().UnsetPrincipalByCliente(testCtx(), uint(1)).Return(nil)
		enderecoRepo.EXPECT().Create(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, e *models.Endereco) error {
			assert.True(t, e.Principal)
			return nil
		})

		_, err := svc.AdicionarEndereco(testCtx(), 1, &req)
		require.NoError(t, err)
	})

	t.Run("erro: UnsetPrincipalByCliente falha e impede a criação", func(t *testing.T) {
		svc, clienteRepo, enderecoRepo := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(&models.Cliente{ID: 1}, nil)
		req := *reqValido
		req.Principal = true

		enderecoRepo.EXPECT().UnsetPrincipalByCliente(testCtx(), uint(1)).Return(errors.New("erro no banco"))

		resultado, err := svc.AdicionarEndereco(testCtx(), 1, &req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "desmarcar endereços principais")
		assert.Nil(t, resultado)
	})

	t.Run("erro: repo.Create falha", func(t *testing.T) {
		svc, clienteRepo, enderecoRepo := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(&models.Cliente{ID: 1}, nil)
		enderecoRepo.EXPECT().Create(testCtx(), gomock.Any()).Return(errors.New("erro no banco"))

		resultado, err := svc.AdicionarEndereco(testCtx(), 1, reqValido)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "erro ao criar endereço")
		assert.Nil(t, resultado)
	})
}

func TestClienteService_ListarEnderecos(t *testing.T) {
	t.Run("sucesso: converte endereços ativos", func(t *testing.T) {
		svc, _, enderecoRepo := novoClienteServiceMock(t)
		enderecos := []models.Endereco{
			{ID: 1, ClienteID: 5, Logradouro: "Rua A", Numero: "10", Principal: true},
			{ID: 2, ClienteID: 5, Logradouro: "Rua B", Numero: "20"},
		}
		enderecoRepo.EXPECT().FindByClienteAtivos(testCtx(), uint(5)).Return(enderecos, nil)

		resultado, err := svc.ListarEnderecos(testCtx(), 5)
		require.NoError(t, err)
		require.Len(t, resultado, 2)
		assert.Equal(t, "Rua A", resultado[0].Logradouro)
		assert.True(t, resultado[0].Principal)
		assert.Equal(t, "Rua B", resultado[1].Logradouro)
	})

	t.Run("erro: propaga do repositório", func(t *testing.T) {
		svc, _, enderecoRepo := novoClienteServiceMock(t)
		enderecoRepo.EXPECT().FindByClienteAtivos(testCtx(), uint(5)).Return(nil, errors.New("erro no banco"))

		_, err := svc.ListarEnderecos(testCtx(), 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "erro ao listar endereços")
	})
}

func TestClienteService_ValidarDocumento(t *testing.T) {
	svc, _, _ := novoClienteServiceMock(t)

	t.Run("CPF válido", func(t *testing.T) {
		tipo, err := svc.ValidarDocumento("529.982.247-25")
		require.NoError(t, err)
		assert.Equal(t, "fisica", tipo)
	})

	t.Run("CNPJ válido", func(t *testing.T) {
		tipo, err := svc.ValidarDocumento("11.222.333/0001-81")
		require.NoError(t, err)
		assert.Equal(t, "juridica", tipo)
	})

	t.Run("CPF inválido", func(t *testing.T) {
		tipo, err := svc.ValidarDocumento("123.456.789-00")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CPF inválido")
		assert.Empty(t, tipo)
	})

	t.Run("CNPJ inválido", func(t *testing.T) {
		tipo, err := svc.ValidarDocumento("11.222.333/0001-99")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CNPJ inválido")
		assert.Empty(t, tipo)
	})

	t.Run("comprimento inválido", func(t *testing.T) {
		tipo, err := svc.ValidarDocumento("12345")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "11 dígitos")
		assert.Empty(t, tipo)
	})
}

func TestClienteService_compararNomes(t *testing.T) {
	svc, _, _ := novoClienteServiceMock(t)

	tests := []struct {
		name         string
		nome1, nome2 string
		expected     bool
	}{
		{name: "nomes idênticos", nome1: "João", nome2: "João", expected: true},
		{name: "case e espaços diferentes", nome1: "  Maria  ", nome2: "maria", expected: true},
		{name: "similaridade alta (MARTHA/MARHTA ~0.96)", nome1: "MARTHA", nome2: "MARHTA", expected: true},
		{name: "nomes diferentes (ana/joao ~0.53)", nome1: "ana", nome2: "joao", expected: false},
		{name: "primeiro vazio", nome1: "", nome2: "joao", expected: false},
		{name: "segundo vazio", nome1: "joao", nome2: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, svc.compararNomes(tt.nome1, tt.nome2))
		})
	}
}

func TestClienteService_AtualizarStatus(t *testing.T) {
	t.Run("erro: cliente não encontrado", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(9)).Return(nil, gorm.ErrRecordNotFound)

		err := svc.AtualizarStatus(testCtx(), 9, "inativo", "motivo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cliente não encontrado")
	})

	t.Run("sucesso: aplica status, motivo e data", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		cliente := &models.Cliente{ID: 9, Status: models.StatusClienteAtivo}
		clienteRepo.EXPECT().FindByID(testCtx(), uint(9)).Return(cliente, nil)
		clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.Equal(t, models.StatusClienteInativo, c.Status)
			assert.Equal(t, "motivo", c.StatusReason)
			assert.NotNil(t, c.StatusUpdatedAt)
			return nil
		})

		require.NoError(t, svc.AtualizarStatus(testCtx(), 9, models.StatusClienteInativo, "motivo"))
	})
}

func TestClienteService_ReativarCliente(t *testing.T) {
	svc, clienteRepo, _ := novoClienteServiceMock(t)
	cliente := &models.Cliente{ID: 4, Status: models.StatusClienteInativo}
	clienteRepo.EXPECT().FindByID(testCtx(), uint(4)).Return(cliente, nil)
	clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
		assert.Equal(t, models.StatusClienteAtivo, c.Status)
		assert.Equal(t, "Reativado após validação", c.StatusReason)
		return nil
	})

	require.NoError(t, svc.ReativarCliente(testCtx(), 4))
}

func TestClienteService_InativarCliente(t *testing.T) {
	svc, clienteRepo, _ := novoClienteServiceMock(t)
	cliente := &models.Cliente{ID: 4, Status: models.StatusClienteAtivo}
	clienteRepo.EXPECT().FindByID(testCtx(), uint(4)).Return(cliente, nil)
	clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).Return(nil)

	require.NoError(t, svc.InativarCliente(testCtx(), 4, "mudança de dono"))
}

func TestClienteService_FindByID(t *testing.T) {
	t.Run("erro: não encontrado", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(99)).Return(nil, gorm.ErrRecordNotFound)

		resultado, err := svc.FindByID(testCtx(), 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cliente não encontrado")
		assert.Nil(t, resultado)
	})

	t.Run("erro: repositório falha", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(99)).Return(nil, errors.New("banco fora"))

		_, err := svc.FindByID(testCtx(), 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "erro ao buscar cliente")
	})

	t.Run("sucesso: converte para DTO", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(2)).Return(&models.Cliente{ID: 2, TenantID: 1, Telefone: "5547999999999", Nome: "Maria", Status: models.StatusClienteAtivo}, nil)

		resultado, err := svc.FindByID(testCtx(), 2)
		require.NoError(t, err)
		assert.Equal(t, uint(2), resultado.ID)
		assert.Equal(t, "Maria", resultado.Nome)
	})
}

func TestClienteService_FindByTelefone(t *testing.T) {
	t.Run("não encontrado: retorna nil, nil", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByTelefone(testCtx(), "5547999999999", uint(1)).Return(nil, gorm.ErrRecordNotFound)

		resultado, err := svc.FindByTelefone(testCtx(), "5547999999999", 1)
		require.NoError(t, err)
		assert.Nil(t, resultado)
	})

	t.Run("erro inesperado: propaga", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByTelefone(testCtx(), "5547999999999", uint(1)).Return(nil, errors.New("banco fora"))

		_, err := svc.FindByTelefone(testCtx(), "5547999999999", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "banco fora")
	})

	t.Run("sucesso: converte para DTO", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByTelefone(testCtx(), "5547999999999", uint(1)).Return(&models.Cliente{ID: 2, TenantID: 1, Telefone: "5547999999999", Nome: "Maria", Status: models.StatusClienteAtivo}, nil)

		resultado, err := svc.FindByTelefone(testCtx(), "5547999999999", 1)
		require.NoError(t, err)
		assert.Equal(t, uint(2), resultado.ID)
	})
}

func TestClienteService_Update(t *testing.T) {
	t.Run("erro: cliente não encontrado", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(nil, gorm.ErrRecordNotFound)

		_, err := svc.Update(testCtx(), 1, &dto.AtualizarClienteRequest{Nome: "Novo"})
		require.Error(t, err)
	})

	t.Run("sucesso: atualiza nome e email", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		cliente := &models.Cliente{ID: 1, Nome: "Antigo", Email: "antigo@teste.com"}
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(cliente, nil)
		clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.Equal(t, "Novo", c.Nome)
			assert.Equal(t, "novo@teste.com", c.Email)
			return nil
		})

		resultado, err := svc.Update(testCtx(), 1, &dto.AtualizarClienteRequest{Nome: "Novo", Email: "novo@teste.com"})
		require.NoError(t, err)
		assert.Equal(t, "Novo", resultado.Nome)
	})

	t.Run("erro: documento inválido não persiste", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(&models.Cliente{ID: 1}, nil)

		_, err := svc.Update(testCtx(), 1, &dto.AtualizarClienteRequest{InscricaoFederal: "123"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "11 dígitos")
	})

	t.Run("sucesso: documento válido é persistido", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		cliente := &models.Cliente{ID: 1}
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(cliente, nil)
		clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.Equal(t, "529.982.247-25", c.InscricaoFederal)
			return nil
		})

		_, err := svc.Update(testCtx(), 1, &dto.AtualizarClienteRequest{InscricaoFederal: "529.982.247-25"})
		require.NoError(t, err)
	})

	t.Run("sucesso: observações vão para o endereço principal", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		cliente := &models.Cliente{
			ID:        1,
			Enderecos: []models.Endereco{{ID: 5, Principal: true, Logradouro: "Rua A"}},
		}
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(cliente, nil)
		clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			require.Len(t, c.Enderecos, 1)
			assert.Equal(t, "entregar no portão", c.Enderecos[0].Observacoes)
			return nil
		})

		_, err := svc.Update(testCtx(), 1, &dto.AtualizarClienteRequest{Observacoes: "entregar no portão"})
		require.NoError(t, err)
	})
}

func TestClienteService_DefinirEnderecoPrincipal(t *testing.T) {
	t.Run("erro: endereço não encontrado", func(t *testing.T) {
		svc, _, enderecoRepo := novoClienteServiceMock(t)
		enderecoRepo.EXPECT().FindByID(testCtx(), uint(10)).Return(nil, gorm.ErrRecordNotFound)

		err := svc.DefinirEnderecoPrincipal(testCtx(), 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "endereço não encontrado")
	})

	t.Run("erro: endereço de outro cliente", func(t *testing.T) {
		svc, _, enderecoRepo := novoClienteServiceMock(t)
		enderecoRepo.EXPECT().FindByID(testCtx(), uint(10)).Return(&models.Endereco{ID: 10, ClienteID: 2}, nil)

		err := svc.DefinirEnderecoPrincipal(testCtx(), 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "não pertence ao cliente")
	})

	t.Run("sucesso: desmarca anteriores e define principal", func(t *testing.T) {
		svc, _, enderecoRepo := novoClienteServiceMock(t)
		endereco := &models.Endereco{ID: 10, ClienteID: 1, Logradouro: "Rua A"}
		enderecoRepo.EXPECT().FindByID(testCtx(), uint(10)).Return(endereco, nil)
		enderecoRepo.EXPECT().UnsetPrincipalByCliente(testCtx(), uint(1)).Return(nil)
		// Update final marca o endereço como principal
		enderecoRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, e *models.Endereco) error {
			assert.True(t, e.Principal)
			return nil
		})

		require.NoError(t, svc.DefinirEnderecoPrincipal(testCtx(), 1, 10))
	})
}

func TestClienteService_AtualizarDocumento(t *testing.T) {
	t.Run("erro: cliente não encontrado", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(nil, gorm.ErrRecordNotFound)

		err := svc.AtualizarDocumento(testCtx(), 1, "529.982.247-25")
		require.Error(t, err)
	})

	t.Run("erro: documento inválido não persiste", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(&models.Cliente{ID: 1}, nil)

		err := svc.AtualizarDocumento(testCtx(), 1, "123")
		require.Error(t, err)
	})

	t.Run("sucesso: documento válido é persistido", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		cliente := &models.Cliente{ID: 1}
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(cliente, nil)
		clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.Equal(t, "529.982.247-25", c.InscricaoFederal)
			return nil
		})

		require.NoError(t, svc.AtualizarDocumento(testCtx(), 1, "529.982.247-25"))
	})
}

func TestClienteService_AtualizarUltimoPedido(t *testing.T) {
	t.Run("erro: cliente não encontrado", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(nil, gorm.ErrRecordNotFound)

		err := svc.AtualizarUltimoPedido(testCtx(), 1)
		require.Error(t, err)
	})

	t.Run("sucesso: atualiza data do último pedido", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		cliente := &models.Cliente{ID: 1}
		clienteRepo.EXPECT().FindByID(testCtx(), uint(1)).Return(cliente, nil)
		clienteRepo.EXPECT().Update(testCtx(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *models.Cliente) error {
			assert.NotNil(t, c.UltimoPedidoAt)
			return nil
		})

		require.NoError(t, svc.AtualizarUltimoPedido(testCtx(), 1))
	})
}

func TestClienteService_FindByTenant(t *testing.T) {
	svc, clienteRepo, _ := novoClienteServiceMock(t)
	clientes := []models.Cliente{
		{ID: 1, TenantID: 1, Nome: "João", Status: models.StatusClienteAtivo},
		{ID: 2, TenantID: 1, Nome: "Maria", Status: models.StatusClienteAtivo},
	}
	clienteRepo.EXPECT().FindByTenant(testCtx(), "1").Return(clientes, nil)

	resultado, err := svc.FindByTenant(testCtx(), 1)
	require.NoError(t, err)
	require.Len(t, resultado, 2)
	assert.Equal(t, "João", resultado[0].Nome)
	assert.Equal(t, "Maria", resultado[1].Nome)
}

func TestClienteService_CountByTenant(t *testing.T) {
	svc, clienteRepo, _ := novoClienteServiceMock(t)
	clienteRepo.EXPECT().CountByTenant(testCtx(), "1").Return(int64(5), nil)

	total, err := svc.CountByTenant(testCtx(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
}

func TestClienteService_ListWithFilters(t *testing.T) {
	t.Run("sucesso: aplica filtro e paginação", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clientes := []models.Cliente{{ID: 1, TenantID: 1, Nome: "João", Status: models.StatusClienteAtivo}}
		clienteRepo.EXPECT().FindWithFilters(testCtx(), uint(1), "joão", "", 10, 0).Return(clientes, int64(1), nil)

		resultado, total, err := svc.ListWithFilters(testCtx(), 1, "joão", "", 1, 10)
		require.NoError(t, err)
		require.Len(t, resultado, 1)
		assert.Equal(t, int64(1), total)
	})

	t.Run("erro: propaga do repositório", func(t *testing.T) {
		svc, clienteRepo, _ := novoClienteServiceMock(t)
		clienteRepo.EXPECT().FindWithFilters(testCtx(), uint(1), "", "", 10, 0).Return(nil, int64(0), errors.New("banco fora"))

		_, _, err := svc.ListWithFilters(testCtx(), 1, "", "", 1, 10)
		require.Error(t, err)
	})
}

func TestClienteService_ConverterParaDTO(t *testing.T) {
	svc, _, _ := novoClienteServiceMock(t)

	t.Run("nil retorna nil", func(t *testing.T) {
		assert.Nil(t, svc.ConverterParaDTO(nil))
	})

	t.Run("converte cliente com endereços", func(t *testing.T) {
		cliente := &models.Cliente{
			ID:         1,
			TenantID:   1,
			Telefone:   "5547999999999",
			Nome:       "João",
			NomePerfil: "João",
			Email:      "joao@teste.com",
			Status:     models.StatusClienteAtivo,
			Enderecos: []models.Endereco{
				{ID: 2, ClienteID: 1, Logradouro: "Rua A", Numero: "10", Principal: true},
			},
		}

		resultado := svc.ConverterParaDTO(cliente)
		require.NotNil(t, resultado)
		assert.Equal(t, uint(1), resultado.ID)
		assert.Equal(t, "João", resultado.Nome)
		assert.Equal(t, models.StatusClienteAtivo, resultado.Status)
		require.Len(t, resultado.Enderecos, 1)
		assert.Equal(t, "Rua A", resultado.Enderecos[0].Logradouro)
		assert.True(t, resultado.Enderecos[0].Principal)
	})
}

func TestClienteService_validateCreateRequest(t *testing.T) {
	svc, _, _ := novoClienteServiceMock(t)

	tests := []struct {
		name    string
		req     *dto.CriarClienteRequest
		wantErr bool
	}{
		{name: "sem tenant", req: &dto.CriarClienteRequest{Telefone: "5547999999999", Nome: "João"}, wantErr: true},
		{name: "sem telefone", req: &dto.CriarClienteRequest{TenantID: 1, Nome: "João"}, wantErr: true},
		{name: "telefone inválido", req: &dto.CriarClienteRequest{TenantID: 1, Telefone: "abc", Nome: "João"}, wantErr: true},
		{name: "válido com nome", req: &dto.CriarClienteRequest{TenantID: 1, Telefone: "5547999999999", Nome: "João"}, wantErr: false},
		{name: "nome vazio usa perfil", req: &dto.CriarClienteRequest{TenantID: 1, Telefone: "5547999999999", NomePerfil: "Perfil"}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateCreateRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.req.Nome == "" {
				assert.Equal(t, "Perfil", tt.req.Nome)
			}
		})
	}
}
