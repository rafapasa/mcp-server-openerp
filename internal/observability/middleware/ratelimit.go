// internal/observability/middleware/ratelimit.go
package middleware

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimiter gerencia o rate limiting
type RateLimiter struct {
	mu         sync.RWMutex
	limits     map[string]*rateLimitEntry
	defaultReq int
	defaultWin time.Duration
	overrides  map[string]rateLimitConfig
}

// rateLimitEntry representa o estado de um cliente
type rateLimitEntry struct {
	count     int
	resetTime time.Time
	mu        sync.Mutex
}

// rateLimitConfig configuração de rate limit
type rateLimitConfig struct {
	requests int
	window   time.Duration
}

// RateLimiterConfig configuração do rate limiter
type RateLimiterConfig struct {
	DefaultRequests int
	DefaultWindow   time.Duration
	Overrides       map[string]rateLimitConfig
}

// NewRateLimiter cria um novo rate limiter
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		limits:     make(map[string]*rateLimitEntry),
		defaultReq: cfg.DefaultRequests,
		defaultWin: cfg.DefaultWindow,
		overrides:  cfg.Overrides,
	}
}

// NewRateLimiterFromEnv cria um rate limiter a partir de variáveis de ambiente
func NewRateLimiterFromEnv() *RateLimiter {
	// Configuração padrão
	defaultReq := 30
	defaultWin := time.Minute

	// Lê do .env
	if req := os.Getenv("RATE_LIMIT_REQUESTS"); req != "" {
		if val, err := strconv.Atoi(req); err == nil && val > 0 {
			defaultReq = val
		}
	}

	if win := os.Getenv("RATE_LIMIT_WINDOW"); win != "" {
		if dur, err := time.ParseDuration(win); err == nil {
			defaultWin = dur
		}
	}

	// Lê overrides
	overrides := make(map[string]rateLimitConfig)
	if overridesEnv := os.Getenv("RATE_LIMIT_OVERRIDES"); overridesEnv != "" {
		for _, override := range strings.Split(overridesEnv, ",") {
			parts := strings.Split(override, ":")
			if len(parts) == 3 {
				tenantID := strings.TrimSpace(parts[0])
				requests, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				window, _ := time.ParseDuration(strings.TrimSpace(parts[2]))
				if tenantID != "" && requests > 0 && window > 0 {
					overrides[tenantID] = rateLimitConfig{
						requests: requests,
						window:   window,
					}
				}
			}
		}
	}

	return NewRateLimiter(RateLimiterConfig{
		DefaultRequests: defaultReq,
		DefaultWindow:   defaultWin,
		Overrides:       overrides,
	})
}

// getConfig retorna a configuração para uma chave
func (rl *RateLimiter) getConfig(key string) rateLimitConfig {
	if cfg, ok := rl.overrides[key]; ok {
		return cfg
	}
	return rateLimitConfig{
		requests: rl.defaultReq,
		window:   rl.defaultWin,
	}
}

// Allow verifica se a requisição é permitida
// Retorna: permitido, tempo de espera em segundos (se não permitido)
func (rl *RateLimiter) Allow(key string) (bool, int64) {
	rl.mu.RLock()
	entry, exists := rl.limits[key]
	rl.mu.RUnlock()

	cfg := rl.getConfig(key)

	if !exists {
		rl.mu.Lock()
		rl.limits[key] = &rateLimitEntry{
			count:     1,
			resetTime: time.Now().Add(cfg.window),
		}
		rl.mu.Unlock()
		return true, 0
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Verifica se a janela expirou
	if time.Now().After(entry.resetTime) {
		entry.count = 1
		entry.resetTime = time.Now().Add(cfg.window)
		return true, 0
	}

	// Verifica se excedeu o limite
	if entry.count >= cfg.requests {
		// Calcula o tempo de espera em segundos
		waitTime := int64(time.Until(entry.resetTime).Seconds())
		if waitTime < 1 {
			waitTime = 1
		}
		return false, waitTime
	}

	entry.count++
	return true, 0
}

// Cleanup remove entradas expiradas (chamado periodicamente)
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, entry := range rl.limits {
		entry.mu.Lock()
		if now.After(entry.resetTime) {
			delete(rl.limits, key)
		}
		entry.mu.Unlock()
	}
}

// GetLimits retorna os limites atuais (para debugging)
func (rl *RateLimiter) GetLimits(key string) (int, time.Time) {
	rl.mu.RLock()
	entry, exists := rl.limits[key]
	rl.mu.RUnlock()

	if !exists {
		return 0, time.Time{}
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()
	return entry.count, entry.resetTime
}
