// internal/observability/logger/redact.go
package logger

import (
	"regexp"
	"strings"

	"go.uber.org/zap"
)

// RedactFields redige campos sensíveis dos logs
func RedactFields(fields []zap.Field) []zap.Field {
	sensitiveKeys := map[string]bool{
		"password": true,
		"token":    true,
		"api_key":  true,
		"secret":   true,
		"auth":     true,
	}

	redacted := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		if sensitiveKeys[strings.ToLower(field.Key)] {
			// Redige o valor
			redacted = append(redacted, zap.String(field.Key, "[REDACTED]"))
		} else {
			redacted = append(redacted, field)
		}
	}
	return redacted
}

// RedactMessage redige mensagens com dados sensíveis
func RedactMessage(msg string) string {
	// Remove tokens e chaves
	tokenRegex := regexp.MustCompile(`(token|key|secret)[=:]\S+`)
	msg = tokenRegex.ReplaceAllString(msg, "$1=[REDACTED]")

	// Remove senhas
	passwordRegex := regexp.MustCompile(`(password|passwd|senha)[=:]\S+`)
	msg = passwordRegex.ReplaceAllString(msg, "$1=[REDACTED]")

	return msg
}
