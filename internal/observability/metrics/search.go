// internal/observability/metrics/search.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SearchTotal total de buscas
	SearchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "search_total",
			Help: "Total number of product searches",
		},
		[]string{"tenant_id", "status"}, // status: found, not_found, error
	)

	// SearchLatency latência das buscas
	SearchLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "search_latency_seconds",
			Help:    "Search latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"tenant_id"},
	)

	// SearchResultsTotal resultados por busca
	SearchResultsTotal = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "search_results_total",
			Help:    "Number of results per search",
			Buckets: []float64{0, 1, 2, 3, 5, 8, 10, 20, 50},
		},
		[]string{"tenant_id"},
	)
)

// SearchRegistered registra uma busca
func SearchRegistered(tenantID, status string, latency float64, results int) {
	SearchTotal.WithLabelValues(tenantID, status).Inc()
	SearchLatency.WithLabelValues(tenantID).Observe(latency)
	SearchResultsTotal.WithLabelValues(tenantID).Observe(float64(results))
}
