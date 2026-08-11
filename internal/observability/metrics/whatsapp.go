// internal/observability/metrics/whatsapp.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WhatsAppMessagesTotal total de mensagens recebidas
	WhatsAppMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whatsapp_messages_total",
			Help: "Total number of WhatsApp messages received",
		},
		[]string{"tenant_id", "status"}, // status: received, processed, error
	)

	// WhatsAppMessagesLatency latência de processamento
	WhatsAppMessagesLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "whatsapp_messages_latency_seconds",
			Help:    "WhatsApp message processing latency in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"tenant_id"},
	)

	// WhatsAppMessagesErrors erros por tipo
	WhatsAppMessagesErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whatsapp_messages_errors_total",
			Help: "Total number of WhatsApp message errors",
		},
		[]string{"tenant_id", "error_type"},
	)

	// WhatsAppActiveSessions sessões ativas
	WhatsAppActiveSessions = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "whatsapp_active_sessions",
			Help: "Number of active WhatsApp sessions",
		},
		[]string{"tenant_id"},
	)
)

// MessageReceived registra uma mensagem recebida
func MessageReceived(tenantID string) {
	WhatsAppMessagesTotal.WithLabelValues(tenantID, "received").Inc()
}

// MessageProcessed registra uma mensagem processada com sucesso
func MessageProcessed(tenantID string, latency float64) {
	WhatsAppMessagesTotal.WithLabelValues(tenantID, "processed").Inc()
	WhatsAppMessagesLatency.WithLabelValues(tenantID).Observe(latency)
}

// MessageError registra um erro no processamento
func MessageError(tenantID, errorType string) {
	WhatsAppMessagesTotal.WithLabelValues(tenantID, "error").Inc()
	WhatsAppMessagesErrors.WithLabelValues(tenantID, errorType).Inc()
}

// SessionActive atualiza sessões ativas
func SessionActive(tenantID string, count float64) {
	WhatsAppActiveSessions.WithLabelValues(tenantID).Set(count)
}
