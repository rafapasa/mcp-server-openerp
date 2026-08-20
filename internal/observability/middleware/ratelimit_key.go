// internal/observability/middleware/ratelimit_key.go
package middleware

import (
	"net/http"
	"strings"
)

// KeyExtractor é uma função que extrai a chave de rate limit da requisição
type KeyExtractor func(r *http.Request) string

// TenantKeyExtractor extrai a chave baseada no tenant_id
func TenantKeyExtractor(r *http.Request) string {
	// Tenta extrair do header
	if tenant := r.Header.Get("X-Tenant-ID"); tenant != "" {
		return "tenant:" + tenant
	}

	// Tenta extrair do query param
	if tenant := r.URL.Query().Get("tenant_id"); tenant != "" {
		return "tenant:" + tenant
	}

	// Tenta extrair do body (apenas para POST)
	if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "application/json" {
		// Se for webhook, usa o número de telefone
		// Implementação simplificada
	}

	// Fallback: IP do cliente
	ip := getClientIP(r)
	return "ip:" + ip
}

// ClientKeyExtractor extrai a chave baseada no cliente (tenant + cliente)
func ClientKeyExtractor(r *http.Request) string {
	// Combina tenant + cliente para rate limit mais granular
	tenant := r.Header.Get("X-Tenant-ID")
	client := r.Header.Get("X-Client-ID")

	if tenant != "" && client != "" {
		return "client:" + tenant + ":" + client
	}

	// Fallback para tenant
	if tenant != "" {
		return "tenant:" + tenant
	}

	// Fallback para IP
	return "ip:" + getClientIP(r)
}

// IPKeyExtractor extrai a chave baseada apenas no IP
func IPKeyExtractor(r *http.Request) string {
	return "ip:" + getClientIP(r)
}

// getClientIP obtém o IP real do cliente (considerando proxies)
func getClientIP(r *http.Request) string {
	// Verifica X-Forwarded-For (quando atrás de proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Verifica X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback: RemoteAddr
	return r.RemoteAddr
}
