package cache

import (
	"context"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/database"
)

// Alias - tudo vem do database.Redis agora
type Cache = database.Redis

func New(r *database.Redis) *Cache {
	return r
}

// GetOrSet expõe o genérico do database
func GetOrSet[T any](c *Cache, ctx context.Context, key string, ttl time.Duration, fn func() (T, error)) (T, error) {
	return database.GetOrSet(c, ctx, key, ttl, fn)
}
