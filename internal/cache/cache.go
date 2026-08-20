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

// GetOrSet - padrão que faltava. Só busca no MySQL se não tiver no Redis
func GetOrSet[T any](c *Cache, ctx context.Context, key string, ttl time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	// 1. Tenta Redis
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(val), &zero); err == nil {
			return zero, nil
		}
	}

	// 2. Miss - busca na fonte (MySQL)
	result, err := fn()
	if err != nil {
		return zero, err
	}

	// 3. Salva no Redis de forma async pra não bloquear resposta
	data, _ := json.Marshal(result)
	_ = c.client.Set(ctx, key, data, ttl).Err()

	return result, nil
}

// Invalidate por padrão de tenant - quando atualiza produto, limpa tudo do tenant
func (c *Cache) InvalidateByTenant(ctx context.Context, tenantID uint, patterns ...string) {
	for _, p := range patterns {
		keys, _ := c.client.Keys(ctx, p).Result()
		if len(keys) > 0 {
			c.client.Del(ctx, keys...)
		}
	}
	// limpa cardápio específico
	c.client.Del(ctx) // use sprintf na chamada real

}

// Para carrinho - mantém seu código, mas com wrapper
func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) bool {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(data), dest) == nil
}

func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, _ := json.Marshal(value)
	return c.client.Set(ctx, key, data, ttl).Err()
}
