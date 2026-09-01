package service

import (
	"context"
	"testing"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeTenantRepo é um repo em memória que registra o tenant criado e permite
// simular FindByID (usado por GetPromptContext).
type fakeTenantRepo struct {
	created       *models.Tenant
	byID          *models.Tenant
	byPhoneID     *models.Tenant
	byVerifyToken *models.Tenant
}

func (f *fakeTenantRepo) FindByID(ctx context.Context, id uint) (*models.Tenant, error) {
	if f.byID == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.byID, nil
}

func (f *fakeTenantRepo) FindByCNPJ(ctx context.Context, cnpj string) (*models.Tenant, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeTenantRepo) FindByTelefone(ctx context.Context, telefone string) (*models.Tenant, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeTenantRepo) FindByWhatsAppPhoneID(ctx context.Context, phoneID string) (*models.Tenant, error) {
	if f.byPhoneID == nil || f.byPhoneID.WhatsappPhoneID != phoneID {
		return nil, gorm.ErrRecordNotFound
	}
	return f.byPhoneID, nil
}

func (f *fakeTenantRepo) FindByVerifyToken(ctx context.Context, token string) (*models.Tenant, error) {
	if f.byVerifyToken == nil || f.byVerifyToken.WhatsappVerifyToken != token {
		return nil, gorm.ErrRecordNotFound
	}
	return f.byVerifyToken, nil
}

func (f *fakeTenantRepo) Create(ctx context.Context, tenant *models.Tenant) error {
	f.created = tenant
	return nil
}

func (f *fakeTenantRepo) Update(ctx context.Context, tenant *models.Tenant) error {
	return nil
}

func (f *fakeTenantRepo) List(ctx context.Context) ([]models.Tenant, error) {
	return nil, nil
}

func TestTenantServiceCreatePersisteSegmento(t *testing.T) {
	tests := []struct {
		name     string
		segmento string
		want     string
	}{
		{name: "segmento informado é repassado ao model", segmento: "farmacia", want: "farmacia"},
		{name: "segmento vazio é repassado como vazio (default fica com o banco)", segmento: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeTenantRepo{}
			svc := NewTenantService(fake, nil)

			out, err := svc.Create(context.Background(), dto.CreateTenantDTO{
				Nome:            "Farmácia Teste",
				CNPJ:            "12.345.678/0001-90",
				Segmento:        tt.segmento,
				WhatsappPhoneID: "987654321098765",
			})
			require.NoError(t, err)
			require.NotNil(t, fake.created)
			assert.Equal(t, tt.want, fake.created.Segmento)
			require.NotNil(t, out)
			assert.Equal(t, tt.want, out.Segmento)
		})
	}
}

func TestTenantServiceGetByVerifyToken(t *testing.T) {
	t.Run("token válido retorna tenant", func(t *testing.T) {
		fake := &fakeTenantRepo{byVerifyToken: &models.Tenant{
			ID:                  3,
			Nome:                "Farmácia Teste",
			WhatsappVerifyToken: "verify-seguro-123",
		}}
		svc := NewTenantService(fake, nil)

		out, err := svc.GetByVerifyToken(context.Background(), "verify-seguro-123")
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, uint(3), out.ID)
	})

	t.Run("token inexistente retorna erro", func(t *testing.T) {
		fake := &fakeTenantRepo{}
		svc := NewTenantService(fake, nil)

		_, err := svc.GetByVerifyToken(context.Background(), "token-desconhecido")
		require.Error(t, err)
	})

	t.Run("token vazio retorna erro", func(t *testing.T) {
		svc := NewTenantService(&fakeTenantRepo{}, nil)

		_, err := svc.GetByVerifyToken(context.Background(), "")
		require.Error(t, err)
	})
}

func TestTenantServiceGetPromptContext(t *testing.T) {
	t.Run("retorna nome e segmento do tenant", func(t *testing.T) {
		fake := &fakeTenantRepo{byID: &models.Tenant{
			ID:       1,
			Nome:     "Mercado Central",
			Segmento: "mercado",
		}}
		svc := NewTenantService(fake, nil)

		nome, segmento, err := svc.GetPromptContext(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, "Mercado Central", nome)
		assert.Equal(t, "mercado", segmento)
	})

	t.Run("segmento vazio cai para geral", func(t *testing.T) {
		fake := &fakeTenantRepo{byID: &models.Tenant{
			ID:   2,
			Nome: "Loja Sem Segmento",
		}}
		svc := NewTenantService(fake, nil)

		_, segmento, err := svc.GetPromptContext(context.Background(), 2)
		require.NoError(t, err)
		assert.Equal(t, "geral", segmento)
	})

	t.Run("tenant inexistente retorna erro", func(t *testing.T) {
		fake := &fakeTenantRepo{}
		svc := NewTenantService(fake, nil)

		_, _, err := svc.GetPromptContext(context.Background(), 99)
		require.Error(t, err)
	})
}
