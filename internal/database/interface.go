package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisInterface define os métodos do Redis usados pela aplicação.
// Permite mocking e futuras substituições de implementação.
type RedisInterface interface {
	// Operações básicas
	Close() error
	Ping() error
	IsConnected() bool
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string) (string, error)
	Delete(keys ...string) error
	Exists(key string) (bool, error)

	// Operações com JSON
	GetJSON(key string, dest interface{}) error
	SetJSON(key string, value interface{}, ttl time.Duration) error
	SetWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetWithContext(ctx context.Context, key string) (string, error)
	GetJSONWithContext(ctx context.Context, key string, dest interface{}) error
	SetJSONWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Operações auxiliares
	DeleteWithContext(ctx context.Context, key string) error
	InvalidateByTenant(ctx context.Context, pattern string) error
	Expire(key string, expiration time.Duration) error

	// GetClient pode ser omitido se não for usado em serviços, mas mantenha se necessário
	GetClient() *redis.Client                       // ou *redis.Client, mas evite dependência concreta
	WithContext(ctx context.Context) RedisInterface // ou *Redis
}
