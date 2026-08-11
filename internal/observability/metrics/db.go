// internal/observability/metrics/db.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DBQueryTotal total de consultas
	DBQueryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_total",
			Help: "Total number of database queries",
		},
		[]string{"table", "operation", "status"}, // operation: select, insert, update, delete
	)

	// DBQueryLatency latência das consultas
	DBQueryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_latency_seconds",
			Help:    "Database query latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"table", "operation"},
	)

	// DBQueryErrors erros por tipo
	DBQueryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Total number of database query errors",
		},
		[]string{"table", "operation", "error_type"},
	)

	// DBPoolConnections conexões no pool
	DBPoolConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_connections",
			Help: "Number of database pool connections",
		},
	)

	// DBPoolIdleConnections conexões idle no pool
	DBPoolIdleConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_idle_connections",
			Help: "Number of idle database pool connections",
		},
	)
)

// DBQuery registra uma consulta
func DBQuery(table, operation, status string, latency float64) {
	DBQueryTotal.WithLabelValues(table, operation, status).Inc()
	if status == "success" {
		DBQueryLatency.WithLabelValues(table, operation).Observe(latency)
	}
}

// DBQueryError registra um erro na consulta
func DBQueryError(table, operation, errorType string) {
	DBQueryErrors.WithLabelValues(table, operation, errorType).Inc()
}

// DBPoolUpdate atualiza métricas do pool
func DBPoolUpdate(connections, idle float64) {
	DBPoolConnections.Set(connections)
	DBPoolIdleConnections.Set(idle)
}
