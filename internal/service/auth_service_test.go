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
