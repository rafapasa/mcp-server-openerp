package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
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

func (s *authService) Authenticate(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.TenantID, req.Email)
	if err != nil {
		return nil, errors.New("credenciais inválidas")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("credenciais inválidas")
	}

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
		return nil, err
	}

	return &dto.LoginResponse{
		Token:   tokenString,
		Expires: exp.Format(time.RFC3339),
		User: dto.UserDTO{
			ID:       user.ID,
			TenantID: user.TenantID,
			Nome:     user.Nome,
			Email:    user.Email,
			Role:     user.Role,
		},
	}, nil
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
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token inválido")
}
