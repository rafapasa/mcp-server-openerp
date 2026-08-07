package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/redis/go-redis/v9"
)

// Redis representa a conexão com o Redis
type Redis struct {
	Client *redis.Client
	ctx    context.Context
}

// NewRedis cria uma nova conexão com o Redis
func NewRedis(cfg *config.Config) (*Redis, error) {
	// Configurar o cliente Redis
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     100,
		MinIdleConns: 10,
		PoolTimeout:  time.Second * 5,
		ReadTimeout:  time.Second * 3,
		WriteTimeout: time.Second * 3,
	})

	// Criar contexto
	ctx := context.Background()

	// Testar a conexão
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("erro ao conectar ao Redis: %w", err)
	}

	log.Println("✅ Conectado ao Redis com sucesso!")
	log.Printf("📊 Redis: %s:%s (DB: %d)", cfg.RedisHost, cfg.RedisPort, cfg.RedisDB)

	return &Redis{
		Client: client,
		ctx:    ctx,
	}, nil
}

// Close fecha a conexão com o Redis
func (r *Redis) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}

// Ping verifica se a conexão está ativa
func (r *Redis) Ping() error {
	return r.Client.Ping(r.ctx).Err()
}

// IsConnected verifica se está conectado ao Redis
func (r *Redis) IsConnected() bool {
	return r.Ping() == nil
}

// Set armazena um valor no Redis com expiração opcional
func (r *Redis) Set(key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(r.ctx, key, value, expiration).Err()
}

// Get recupera um valor do Redis
func (r *Redis) Get(key string) (string, error) {
	return r.Client.Get(r.ctx, key).Result()
}

// Delete remove uma ou mais chaves do Redis
func (r *Redis) Delete(keys ...string) error {
	return r.Client.Del(r.ctx, keys...).Err()
}

// Exists verifica se uma chave existe no Redis
func (r *Redis) Exists(key string) (bool, error) {
	result, err := r.Client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Expire define um tempo de expiração para uma chave
func (r *Redis) Expire(key string, expiration time.Duration) error {
	return r.Client.Expire(r.ctx, key, expiration).Err()
}

// TTL retorna o tempo restante de expiração de uma chave
func (r *Redis) TTL(key string) (time.Duration, error) {
	return r.Client.TTL(r.ctx, key).Result()
}

// Increment incrementa um valor inteiro no Redis
func (r *Redis) Increment(key string) (int64, error) {
	return r.Client.Incr(r.ctx, key).Result()
}

// Decrement decrementa um valor inteiro no Redis
func (r *Redis) Decrement(key string) (int64, error) {
	return r.Client.Decr(r.ctx, key).Result()
}

// HSet define um campo em um hash no Redis
func (r *Redis) HSet(key string, values ...interface{}) error {
	return r.Client.HSet(r.ctx, key, values...).Err()
}

// HGet recupera um campo de um hash no Redis
func (r *Redis) HGet(key, field string) (string, error) {
	return r.Client.HGet(r.ctx, key, field).Result()
}

// HGetAll recupera todos os campos de um hash no Redis
func (r *Redis) HGetAll(key string) (map[string]string, error) {
	return r.Client.HGetAll(r.ctx, key).Result()
}

// HDel remove campos de um hash no Redis
func (r *Redis) HDel(key string, fields ...string) error {
	return r.Client.HDel(r.ctx, key, fields...).Err()
}

// LPush adiciona um ou mais valores ao início de uma lista
func (r *Redis) LPush(key string, values ...interface{}) error {
	return r.Client.LPush(r.ctx, key, values...).Err()
}

// RPush adiciona um ou mais valores ao final de uma lista
func (r *Redis) RPush(key string, values ...interface{}) error {
	return r.Client.RPush(r.ctx, key, values...).Err()
}

// LPop remove e retorna o primeiro elemento de uma lista
func (r *Redis) LPop(key string) (string, error) {
	return r.Client.LPop(r.ctx, key).Result()
}

// RPop remove e retorna o último elemento de uma lista
func (r *Redis) RPop(key string) (string, error) {
	return r.Client.RPop(r.ctx, key).Result()
}

// SAdd adiciona membros a um conjunto
func (r *Redis) SAdd(key string, members ...interface{}) error {
	return r.Client.SAdd(r.ctx, key, members...).Err()
}

// SMembers retorna todos os membros de um conjunto
func (r *Redis) SMembers(key string) ([]string, error) {
	return r.Client.SMembers(r.ctx, key).Result()
}

// SRem remove membros de um conjunto
func (r *Redis) SRem(key string, members ...interface{}) error {
	return r.Client.SRem(r.ctx, key, members...).Err()
}

// ZAdd adiciona membros a um conjunto ordenado
func (r *Redis) ZAdd(key string, members ...*redis.Z) error {
	// Convert []*redis.Z to []redis.Z as required by go-redis/v9 ZAdd method
	zMembers := make([]redis.Z, len(members))
	for i, m := range members {
		zMembers[i] = *m
	}
	return r.Client.ZAdd(r.ctx, key, zMembers...).Err()
}

// ZRange retorna membros de um conjunto ordenado por intervalo
func (r *Redis) ZRange(key string, start, stop int64) ([]string, error) {
	return r.Client.ZRange(r.ctx, key, start, stop).Result()
}

// FlushAll limpa todos os dados do Redis (cuidado em produção!)
func (r *Redis) FlushAll() error {
	return r.Client.FlushAll(r.ctx).Err()
}

// GetClient retorna a instância do cliente Redis
func (r *Redis) GetClient() *redis.Client {
	return r.Client
}

// WithContext permite definir um contexto customizado
func (r *Redis) WithContext(ctx context.Context) *Redis {
	return &Redis{
		Client: r.Client,
		ctx:    ctx,
	}
}
