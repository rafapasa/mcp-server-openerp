// internal/observability/middleware/security.go
package middleware

import (
	"net/http"
	"os"
	"strings"
)

// SecurityHeadersMiddleware adiciona headers de segurança em todas as respostas
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. HSTS - Força HTTPS (apenas em produção)
		if IsProduction() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// 2. X-Content-Type-Options - Previne MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// 3. X-Frame-Options - Previne clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// 4. X-XSS-Protection - Proteção XSS (legado, mas ainda útil)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// 5. Referrer-Policy - Controla envio de referrer
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 6. Permissions-Policy - Limita APIs do navegador
		w.Header().Set("Permissions-Policy",
			"geolocation=(), microphone=(), camera=(), payment=(), usb=(), "+
				"magnetometer=(), gyroscope=(), accelerometer=(), "+
				"document-domain=()",
		)

		// 7. Content-Security-Policy - Controla recursos carregados
		csp := buildCSP()
		if csp != "" {
			w.Header().Set("Content-Security-Policy", csp)
		}

		// 8. Remove headers que revelam informações do servidor
		w.Header().Del("Server")
		w.Header().Del("X-Powered-By")

		next.ServeHTTP(w, r)
	})
}

// isProduction verifica se estamos em ambiente de produção
func IsProduction() bool {
	env := os.Getenv("ENVIRONMENT")
	return env == "production" || env == "prod"
}

// buildCSP constrói a política de segurança de conteúdo
func buildCSP() string {
	// Define as diretivas base
	directives := []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: https:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"frame-src 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}

	// Se estiver em desenvolvimento, adiciona exceções
	if os.Getenv("ENVIRONMENT") == "development" {
		directives = append(directives, "script-src 'self' 'unsafe-inline' 'unsafe-eval'")
	}

	return strings.Join(directives, "; ")
}
