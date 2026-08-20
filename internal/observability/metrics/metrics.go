package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal   *prometheus.CounterVec
	HttpRequestDuration *prometheus.HistogramVec
	WebhookEventsTotal  *prometheus.CounterVec
)

func Init() {
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total de requisições HTTP",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duração das requisições HTTP",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	WebhookEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_events_total",
			Help: "Total de eventos webhook",
		},
		[]string{"type", "tenant_id"},
	)
}
