package service

import (
	"context"
	"testing"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type authUserRepositoryStub struct {
	users []models.User
	err   error
}

func (r *authUserRepositoryStub) FindByEmail(context.Context, uint, string) (*[]models.User, error) {
	return &r.users, r.err
}

func (r *authUserRepositoryStub) FindByID(context.Context, uint) (*models.User, error) {
	return nil, nil
}

func (r *authUserRepositoryStub) FindByTenantPaginated(context.Context, uint, int, int) ([]models.User, int64, error) {
	return r.users, int64(len(r.users)), r.err
}

func (r *authUserRepositoryStub) Create(context.Context, *models.User) error {
	return nil
}

func TestAuthServiceAuthenticate(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	t.Run("autentica senha válida e gera token", func(t *testing.T) {
		repo := &authUserRepositoryStub{users: []models.User{{
			ID: 1, TenantID: 1, Nome: "Admin", Email: "etoolstec@etoolstec.com.br",
			PasswordHash: string(passwordHash), Role: "admin", IsActive: true,
		}}}
		svc := NewAuthService(repo, &config.Config{JWTSecret: "test-secret"})

		logins, err := svc.Authenticate(context.Background(), dto.LoginRequest{
			Email: "etoolstec@etoolstec.com.br", Password: "admin123", TenantID: 1,
		})

		for _, response := range logins.Users {
			require.NoError(t, err)
			require.NotEmpty(t, response.Token)
			require.Equal(t, uint(1), response.User.ID)
			require.Equal(t, "etoolstec@etoolstec.com.br", response.User.Email)
			claims, err := svc.ValidateToken(response.Token)
			require.NoError(t, err)
			require.Equal(t, uint(1), claims.UserID)
			require.Equal(t, uint(1), claims.TenantID)
		}
	})

	t.Run("rejeita senha incorreta", func(t *testing.T) {
		repo := &authUserRepositoryStub{users: []models.User{{PasswordHash: string(passwordHash)}}}
		svc := NewAuthService(repo, &config.Config{JWTSecret: "test-secret"})

		_, err := svc.Authenticate(context.Background(), dto.LoginRequest{
			Email: "etoolstec@etoolstec.com.br", Password: "senha-incorreta", TenantID: 1,
		})

		require.EqualError(t, err, "credenciais inválidas")
	})

	t.Run("rejeita usuário inexistente", func(t *testing.T) {
		svc := NewAuthService(&authUserRepositoryStub{}, &config.Config{JWTSecret: "test-secret"})

		_, err := svc.Authenticate(context.Background(), dto.LoginRequest{
			Email: "etoolstec@etoolstec.com.br", Password: "admin123", TenantID: 1,
		})

		require.EqualError(t, err, "credenciais inválidas, usuário não encontrado")
	})

	t.Run("rejeita token adulterado", func(t *testing.T) {
		svc := NewAuthService(&authUserRepositoryStub{}, &config.Config{JWTSecret: "test-secret"})

		_, err := svc.ValidateToken("token.adulterado")

		require.Error(t, err)
	})
}

func TestAuthServiceFindByTenantPaginated(t *testing.T) {
	t.Run("erro: tenant_id não informado", func(t *testing.T) {
		svc := NewAuthService(&authUserRepositoryStub{}, &config.Config{JWTSecret: "test-secret"})

		users, total, err := svc.FindByTenantPaginated(context.Background(), 0, 1, 10)
		require.Error(t, err)
		require.Nil(t, users)
		require.Zero(t, total)
	})

	t.Run("sucesso: converte usuários para DTO", func(t *testing.T) {
		repo := &authUserRepositoryStub{users: []models.User{
			{ID: 1, TenantID: 1, Nome: "Admin", Email: "admin@teste.com", Role: "admin"},
			{ID: 2, TenantID: 1, Nome: "Operador", Email: "op@teste.com", Role: "user"},
		}}
		svc := NewAuthService(repo, &config.Config{JWTSecret: "test-secret"})

		users, total, err := svc.FindByTenantPaginated(context.Background(), 1, 1, 10)
		require.NoError(t, err)
		require.Len(t, users, 2)
		require.Equal(t, int64(2), total)
		require.Equal(t, uint(1), users[0].ID)
		require.Equal(t, "Admin", users[0].Nome)
		require.Equal(t, "admin", users[0].Role)
	})

	t.Run("erro: propaga do repositório", func(t *testing.T) {
		repo := &authUserRepositoryStub{err: context.DeadlineExceeded}
		svc := NewAuthService(repo, &config.Config{JWTSecret: "test-secret"})

		_, _, err := svc.FindByTenantPaginated(context.Background(), 1, 1, 10)
		require.Error(t, err)
	})
}
