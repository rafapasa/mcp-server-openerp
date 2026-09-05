package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/redis/go-redis/v9"
)

type redisClient struct {
	Client *redis.Client
	ctx    context.Context
}

// NewRedis conecta ao Redis e retorna sua interface de acesso.
func NewRedis(cfg *config.Config) (RedisInterface, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     100,
		MinIdleConns: 10,
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("erro ao conectar ao Redis: %w", err)
	}
	logger.GetLogger().Info("✅ Redis conectado")
	logger.GetLogger().Info(fmt.Sprintf("📊 Redis: %s:%s (DB: %d)", cfg.RedisHost, cfg.RedisPort, cfg.RedisDB))
	return &redisClient{Client: client, ctx: ctx}, nil
}

func (r *redisClient) Close() error      { return r.Client.Close() }
func (r *redisClient) Ping() error       { return r.Client.Ping(r.ctx).Err() }
func (r *redisClient) IsConnected() bool { return r.Ping() == nil }

func (r *redisClient) Set(key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(r.ctx, key, value, expiration).Err()
}

func (r *redisClient) Get(key string) (string, error) {
	return r.Client.Get(r.ctx, key).Result()
}
func (r *redisClient) Delete(keys ...string) error { return r.Client.Del(r.ctx, keys...).Err() }
func (r *redisClient) Exists(key string) (bool, error) {
	n, err := r.Client.Exists(r.ctx, key).Result()
	return n > 0, err
}

// Deduplicação atômica
func (r *redisClient) SetNX(key string, value interface{}, ttl time.Duration) (bool, error) {
	return r.Client.SetNX(r.ctx, key, value, ttl).Result()
}

func (r *redisClient) SetNXWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return r.Client.SetNX(ctx, key, value, ttl).Result()
}

func (r *redisClient) GetJSON(key string, dest interface{}) error {
	if r == nil || r.Client == nil {
		return redis.Nil
	}
	data, err := r.Client.Get(r.ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (r *redisClient) SetJSON(key string, value interface{}, ttl time.Duration) error {
	if r == nil || r.Client == nil {
		return nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(r.ctx, key, b, ttl).Err()
}

func (r *redisClient) SetWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.Client.Set(ctx, key, value, ttl).Err()
}

func (r *redisClient) GetWithContext(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

func (r *redisClient) GetJSONWithContext(ctx context.Context, key string, dest interface{}) error {
	data, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (r *redisClient) SetJSONWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, key, b, ttl).Err()
}

// GetOrSet retorna o valor em cache ou executa a função para carregá-lo.
func GetOrSet[T any](r RedisInterface, ctx context.Context, key string, ttl time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	if r == nil || r.GetClient() == nil {
		return fn()
	}
	var cached T
	if err := r.GetJSONWithContext(ctx, key, &cached); err == nil {
		return cached, nil
	}
	val, err := fn()
	if err != nil {
		return zero, err
	}
	_ = r.SetJSONWithContext(ctx, key, val, ttl)
	return val, nil
}

func (r *redisClient) InvalidateByTenant(ctx context.Context, pattern string) error {
	keys, err := r.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return r.Client.Del(ctx, keys...).Err()
	}
	return nil
}

func (r *redisClient) DeleteWithContext(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

func (r *redisClient) Expire(key string, expiration time.Duration) error {
	return r.Client.Expire(r.ctx, key, expiration).Err()
}
func (r *redisClient) GetClient() *redis.Client { return r.Client }
func (r *redisClient) WithContext(ctx context.Context) RedisInterface {
	return &redisClient{Client: r.Client, ctx: ctx}
}
