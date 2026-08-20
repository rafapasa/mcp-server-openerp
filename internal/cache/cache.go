// internal/cache/cache.go - COMPLETO com Get, Set, GetJSON, SetJSON
package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func New(client *redis.Client) *Cache {
	return &Cache{client: client}
}

// Get simples - retorna string
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	if c == nil || c.client == nil {
		return "", redis.Nil
	}
	return c.client.Get(ctx, key).Result()
}

// Set simples
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

// GetJSON - busca JSON e deserializa
func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	if c == nil || c.client == nil {
		return redis.Nil
	}
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// SetJSON - serializa e salva
func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, b, ttl).Err()
}

// GetOrSet genérico
func GetOrSet[T any](c *Cache, ctx context.Context, key string, ttl time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	if c == nil || c.client == nil {
		return fn()
	}
	// tenta cache
	var cached T
	err := c.GetJSON(ctx, key, &cached)
	if err == nil {
		return cached, nil
	}
	// miss - busca origem
	val, err := fn()
	if err != nil {
		return zero, err
	}
	_ = c.SetJSON(ctx, key, val, ttl) // best effort
	return val, nil
}

func (c *Cache) InvalidateByTenant(ctx context.Context, tenantID string, pattern string) error {
	if c == nil || c.client == nil {
		return nil
	}
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, key).Err()
}
