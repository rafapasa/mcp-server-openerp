// internal/api/middleware.go
package api

import (
	"context"
	"net/http"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	TenantIDKey contextKey = "tenant_id"
	EmailKey    contextKey = "email"
)

// AuthMiddleware verifica o token JWT
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Pega o header Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		// Extrai o token
		tokenString := ExtractTokenFromHeader(authHeader)
		if tokenString == "" {
			writeError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		// Valida o token
		claims, err := ValidateJWT(tokenString)
		if err != nil {
			logger.Warn(r.Context(), "Invalid JWT", zap.Error(err))
			writeError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Adiciona claims ao contexto
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, TenantIDKey, claims.TenantID)
		ctx = context.WithValue(ctx, EmailKey, claims.Email)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// OptionalAuthMiddleware permite autenticação opcional (para login)
func OptionalAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			tokenString := ExtractTokenFromHeader(authHeader)
			if tokenString != "" {
				claims, err := ValidateJWT(tokenString)
				if err == nil {
					ctx := r.Context()
					ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
					ctx = context.WithValue(ctx, TenantIDKey, claims.TenantID)
					ctx = context.WithValue(ctx, EmailKey, claims.Email)
					r = r.WithContext(ctx)
				}
			}
		}
		next.ServeHTTP(w, r)
	}
}

// GetUserID extrai o user_id do contexto
func GetUserID(r *http.Request) (uint, bool) {
	val := r.Context().Value(UserIDKey)
	if val == nil {
		return 0, false
	}
	id, ok := val.(uint)
	return id, ok
}

// GetTenantID extrai o tenant_id do contexto
func GetTenantID(r *http.Request) (uint, bool) {
	val := r.Context().Value(TenantIDKey)
	if val == nil {
		return 0, false
	}
	id, ok := val.(uint)
	return id, ok
}
