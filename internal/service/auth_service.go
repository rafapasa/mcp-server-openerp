package service

import (
	"context"
	"time"

	"github.com/etoolstec/gokit/apperror"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	TenantID uint   `json:"tenant_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type authService struct {
	userRepo repository.UserRepositoryInterface
	cfg      *config.Config
}

func NewAuthService(userRepo repository.UserRepositoryInterface, cfg *config.Config) AuthServiceInterface {
	return &authService{userRepo: userRepo, cfg: cfg}
}

func (s *authService) Authenticate(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponseList, error) {
	users, err := s.userRepo.FindByEmail(ctx, req.TenantID, req.Email)
	if err != nil {
		return nil, err
	}

	if users == nil || len(*users) == 0 {
		return nil, apperror.NewUnauthorizedError("credenciais inválidas, usuário não encontrado")
	}

	logins := &dto.LoginResponseList{}
	for _, user := range *users {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err == nil {
			exp := time.Now().Add(24 * time.Hour)
			claims := Claims{
				UserID:   user.ID,
				TenantID: user.TenantID,
				Email:    user.Email,
				Role:     user.Role,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(exp),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					NotBefore: jwt.NewNumericDate(time.Now()),
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			secret := []byte(s.cfg.JWTSecret)
			if len(secret) == 0 {
				secret = []byte("default-secret-change-me")
			}
			tokenString, err := token.SignedString(secret)
			if err != nil {
				return nil, apperror.NewInternalError("falha ao gerar token", err)
			}

			logins.Count++
			logins.Users = append(
				logins.Users, dto.LoginResponse{
					Token:   tokenString,
					Expires: exp.Format(time.RFC3339),
					User: dto.UserDTO{
						ID:       user.ID,
						TenantID: user.TenantID,
						Nome:     user.Nome,
						Email:    user.Email,
						Role:     user.Role,
					},
				},
			)
		}
	}
	if len(logins.Users) <= 0 {
		return nil, apperror.NewUnauthorizedError("credenciais inválidas")
	}
	return logins, nil
}

func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
	secret := []byte(s.cfg.JWTSecret)
	if len(secret) == 0 {
		secret = []byte("default-secret-change-me")
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, apperror.NewUnauthorizedError("token inválido: " + err.Error())
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, apperror.NewUnauthorizedError("token inválido")
}

// FindByTenantPaginated lista os usuários de um tenant de forma paginada.
func (s *authService) FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]dto.UserDTO, int64, error) {
	if tenantID == 0 {
		return nil, 0, apperror.NewBadRequestError("tenant_id não informado")
	}
	users, total, err := s.userRepo.FindByTenantPaginated(ctx, tenantID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	result := make([]dto.UserDTO, len(users))
	for i, u := range users {
		result[i] = toUserDTO(&u)
	}
	return result, total, nil
}

func toUserDTO(u *models.User) dto.UserDTO {
	return dto.UserDTO{
		ID:       u.ID,
		TenantID: u.TenantID,
		Nome:     u.Nome,
		Email:    u.Email,
		Role:     u.Role,
	}
}
