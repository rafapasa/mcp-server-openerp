// internal/observability/middleware/sanitize.go
package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/security"
	"go.uber.org/zap"
)

// SanitizeMiddleware sanitiza os dados da requisição
func SanitizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Só sanitiza POST com JSON
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "application/json" {
			body, err := io.ReadAll(r.Body)
			if err == nil {
				// Sanitiza o body
				sanitizedBody := sanitizeJSON(body)
				r.Body = io.NopCloser(bytes.NewReader(sanitizedBody))
			}
		}

		// Sanitiza headers
		sanitizeHeaders(r)

		next.ServeHTTP(w, r)
	})
}

// sanitizeJSON sanitiza campos de texto em JSON
func sanitizeJSON(body []byte) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body
	}

	// Lista de campos que devem ser sanitizados
	textFields := []string{"mensagem", "message", "text", "body", "observacao", "observations"}

	for _, field := range textFields {
		if value, ok := data[field]; ok {
			if str, ok := value.(string); ok {
				data[field] = security.SanitizeMessage(str)
			}
		}
	}

	sanitized, _ := json.Marshal(data)
	return sanitized
}

// sanitizeHeaders sanitiza headers que podem conter dados sensíveis
func sanitizeHeaders(r *http.Request) {
	// Sanitiza header de tenant
	if tenant := r.Header.Get("X-Tenant-ID"); tenant != "" {
		if err := security.ValidateTenantID(tenant); err != nil {
			logger.Warn(r.Context(), "Tenant ID inválido",
				zap.String("tenant_id", tenant),
				zap.Error(err),
			)
			// Não bloqueia a requisição, apenas loga
		}
	}
}
