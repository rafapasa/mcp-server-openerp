// internal/observability/metrics/llm.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// LLMRequestsTotal total de chamadas LLM
	LLMRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_requests_total",
			Help: "Total number of LLM requests",
		},
		[]string{"provider", "model", "status"}, // status: success, error
	)

	// LLMRequestsLatency latência das chamadas LLM
	LLMRequestsLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_requests_latency_seconds",
			Help:    "LLM request latency in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 30, 60},
		},
		[]string{"provider", "model"},
	)

	// LLMTokensUsed tokens usados por chamada
	LLMTokensUsed = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_tokens_used",
			Help:    "Number of tokens used per LLM request",
			Buckets: []float64{50, 100, 200, 500, 1000, 2000, 5000, 10000},
		},
		[]string{"provider", "model", "type"}, // type: prompt, completion
	)

	// LLMCostEstimated custo estimado por chamada
	LLMCostEstimated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_cost_estimated_usd",
			Help: "Estimated cost of LLM requests in USD",
		},
		[]string{"provider", "model"},
	)

	// LLMIntentDistribution distribuição de intenções
	LLMIntentDistribution = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_intent_distribution_total",
			Help: "Distribution of detected intents",
		},
		[]string{"intent"},
	)
	HandoffStarted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "handoff_iniciado_total",
			Help: "Total de handoffs do bot para atendimento humano",
		},
	)
)

// LLMRequest registra uma chamada LLM
func LLMRequest(provider, model, status string, latency float64) {
	LLMRequestsTotal.WithLabelValues(provider, model, status).Inc()
	if status == "success" {
		LLMRequestsLatency.WithLabelValues(provider, model).Observe(latency)
	}
}

func RegisterHandoffStarted() {
	HandoffStarted.Inc()
}

// LLMTokens registra tokens usados
func LLMTokens(provider, model, tokenType string, tokens float64) {
	LLMTokensUsed.WithLabelValues(provider, model, tokenType).Observe(tokens)
}

// LLMCost registra custo estimado
func LLMCost(provider, model string, cost float64) {
	LLMCostEstimated.WithLabelValues(provider, model).Add(cost)
}

// IntentDetected registra uma intenção detectada
func IntentDetected(intent string) {
	LLMIntentDistribution.WithLabelValues(intent).Inc()
}
