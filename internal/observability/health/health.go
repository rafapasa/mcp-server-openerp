// internal/observability/health/health.go
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/rafapasa/mcp-server-openerp/internal/llm"
)

// Status representa o status de um componente
type Status string

const (
	// StatusUp componente está funcionando
	StatusUp Status = "up"
	// StatusDown componente está com problema
	StatusDown Status = "down"
	// StatusDegraded componente está com performance degradada
	StatusDegraded Status = "degraded"
)

// CheckResult representa o resultado de uma verificação
type CheckResult struct {
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Latency   time.Duration `json:"latency_ms"`
	Details   interface{}   `json:"details,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}

// ComponentCheck é uma função que verifica um componente
type ComponentCheck func(ctx context.Context) CheckResult

// HealthChecker gerencia os health checks
type HealthChecker struct {
	mu         sync.RWMutex
	checks     map[string]ComponentCheck
	results    map[string]CheckResult
	lastUpdate time.Time
}

// NewHealthChecker cria um novo checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks:  make(map[string]ComponentCheck),
		results: make(map[string]CheckResult),
	}
}

// Register registra uma verificação para um componente
func (h *HealthChecker) Register(name string, check ComponentCheck) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = check
}

// CheckAll executa todas as verificações
func (h *HealthChecker) CheckAll(ctx context.Context) map[string]CheckResult {
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, check := range h.checks {
		h.results[name] = check(ctx)
	}
	h.lastUpdate = time.Now()

	return h.results
}

// GetResults retorna os resultados atuais
func (h *HealthChecker) GetResults() map[string]CheckResult {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.results
}

// GetStatus retorna o status geral
func (h *HealthChecker) GetStatus() Status {
	results := h.GetResults()
	if len(results) == 0 {
		return StatusDown
	}

	for _, result := range results {
		if result.Status == StatusDown {
			return StatusDown
		}
	}

	for _, result := range results {
		if result.Status == StatusDegraded {
			return StatusDegraded
		}
	}

	return StatusUp
}

// HealthResponse representa a resposta do health check
type HealthResponse struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Checks    map[string]CheckResult `json:"checks"`
	Version   string                 `json:"version"`
	Service   string                 `json:"service"`
}

// NewDefaultHealthChecker cria um checker com verificações padrão
func NewDefaultHealthChecker(db *gorm.DB, redis *redis.Client, llmClient llm.LLMClient) *HealthChecker {
	hc := NewHealthChecker()

	// Verificação do banco de dados
	hc.Register("database", func(ctx context.Context) CheckResult {
		start := time.Now()

		sqlDB, err := db.DB()
		if err != nil {
			return CheckResult{
				Status:    StatusDown,
				Message:   fmt.Sprintf("Erro ao obter DB: %v", err),
				Latency:   time.Since(start),
				CheckedAt: time.Now(),
			}
		}

		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			return CheckResult{
				Status:    StatusDown,
				Message:   fmt.Sprintf("Erro ao pingar DB: %v", err),
				Latency:   time.Since(start),
				CheckedAt: time.Now(),
			}
		}

		// Verifica stats do pool
		stats := sqlDB.Stats()
		details := map[string]interface{}{
			"max_open_connections": stats.MaxOpenConnections,
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
		}

		status := StatusUp
		if stats.OpenConnections > stats.MaxOpenConnections/2 {
			status = StatusDegraded
		}

		return CheckResult{
			Status:    status,
			Message:   "Database OK",
			Latency:   time.Since(start),
			Details:   details,
			CheckedAt: time.Now(),
		}
	})

	// Verificação do Redis
	hc.Register("redis", func(ctx context.Context) CheckResult {
		start := time.Now()

		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := redis.Ping(ctx).Err(); err != nil {
			return CheckResult{
				Status:    StatusDown,
				Message:   fmt.Sprintf("Erro ao pingar Redis: %v", err),
				Latency:   time.Since(start),
				CheckedAt: time.Now(),
			}
		}

		return CheckResult{
			Status:    StatusUp,
			Message:   "Redis OK",
			Latency:   time.Since(start),
			CheckedAt: time.Now(),
		}
	})

	// Verificação do LLM
	hc.Register("llm", func(ctx context.Context) CheckResult {
		start := time.Now()

		if llmClient == nil {
			return CheckResult{
				Status:    StatusDown,
				Message:   "LLM client não configurado",
				Latency:   time.Since(start),
				CheckedAt: time.Now(),
			}
		}

		// Testa o LLM com um prompt simples
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Faz uma chamada simples para testar
		_, err := llmClient.GenerateWithContext(ctx, "Teste de saúde: responda OK")
		if err != nil {
			return CheckResult{
				Status:    StatusDegraded,
				Message:   fmt.Sprintf("LLM com erro: %v", err),
				Latency:   time.Since(start),
				CheckedAt: time.Now(),
			}
		}

		return CheckResult{
			Status:  StatusUp,
			Message: fmt.Sprintf("LLM OK (%s)", llmClient.GetProvider()),
			Latency: time.Since(start),
			Details: map[string]string{
				"provider": llmClient.GetProvider(),
				"model":    llmClient.GetModel(),
			},
			CheckedAt: time.Now(),
		}
	})

	return hc
}

// ============================================
// HANDLERS HTTP
// ============================================

// HealthHandler retorna o status de saúde
func HealthHandler(hc *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Executa verificações
		results := hc.CheckAll(ctx)
		status := hc.GetStatus()

		response := HealthResponse{
			Status:    status,
			Timestamp: time.Now(),
			Uptime:    time.Since(startTime).String(),
			Checks:    results,
			Version:   "1.0.0",
			Service:   "mcp-server",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if status == StatusDown {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(response)
	}
}

// ReadinessHandler retorna se o serviço está pronto para receber tráfego
func ReadinessHandler(hc *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results := hc.GetResults()

		// Verifica se todos os componentes essenciais estão up
		essential := []string{"database", "redis"}
		for _, name := range essential {
			if result, ok := results[name]; ok {
				if result.Status == StatusDown {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					json.NewEncoder(w).Encode(map[string]string{
						"status": "not ready",
						"reason": name + " is down",
					})
					return
				}
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{
					"status": "not ready",
					"reason": name + " not checked yet",
				})
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	}
}

// StatusHandler retorna um status detalhado (para diagnóstico)
func StatusHandler(hc *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Força uma verificação completa
		results := hc.CheckAll(ctx)
		status := hc.GetStatus()

		response := HealthResponse{
			Status:    status,
			Timestamp: time.Now(),
			Uptime:    time.Since(startTime).String(),
			Checks:    results,
			Version:   "1.0.0",
			Service:   "mcp-server",
		}

		w.Header().Set("Content-Type", "application/json")

		if status == StatusDown {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(w).Encode(response)
	}
}

// startTime é o tempo de início do serviço
var startTime = time.Now()
