// internal/observability/metrics/cart.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CartItemsTotal itens por carrinho
	CartItemsTotal = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "cart_items_total",
			Help:    "Number of items in cart",
			Buckets: []float64{1, 2, 3, 5, 8, 10, 15, 20, 30, 50},
		},
	)

	// CartValueTotal valor total do carrinho
	CartValueTotal = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "cart_value_total_brl",
			Help:    "Total value of cart in BRL",
			Buckets: []float64{10, 25, 50, 100, 200, 500, 1000},
		},
	)

	// CartAbandoned carrinhos abandonados
	CartAbandoned = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cart_abandoned_total",
			Help: "Total number of abandoned carts",
		},
	)

	// CartConversionRate taxa de conversão (pedidos/carrinhos)
	CartConversionRate = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cart_conversion_rate",
			Help: "Cart conversion rate (orders/carts)",
		},
	)

	// CartItemsAdded itens adicionados ao carrinho
	CartItemsAdded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cart_items_added_total",
			Help: "Total number of items added to cart",
		},
		[]string{"tenant_id"},
	)

	// CartItemsRemoved itens removidos do carrinho
	CartItemsRemoved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cart_items_removed_total",
			Help: "Total number of items removed from cart",
		},
		[]string{"tenant_id"},
	)
)

// CartUpdated atualiza métricas do carrinho
func CartUpdated(items int, value float64) {
	CartItemsTotal.Observe(float64(items))
	CartValueTotal.Observe(value)
}

// CartAbandonedCount incrementa carrinhos abandonados
func CartAbandonedCount() {
	CartAbandoned.Inc()
}

// CartConversion atualiza taxa de conversão
func CartConversion(rate float64) {
	CartConversionRate.Set(rate)
}

// CartItemAdded registra item adicionado
func CartItemAdded(tenantID string) {
	CartItemsAdded.WithLabelValues(tenantID).Inc()
}

// CartItemRemoved registra item removido
func CartItemRemoved(tenantID string) {
	CartItemsRemoved.WithLabelValues(tenantID).Inc()
}
