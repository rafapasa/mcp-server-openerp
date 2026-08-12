// internal/observability/security/sanitize.go
package security

import (
	"regexp"
	"strings"
)

// SanitizeInput sanitiza uma string de input
// Remove caracteres perigosos e normaliza espaços
func SanitizeInput(input string) string {
	if input == "" {
		return ""
	}

	// 1. Remove espaços extras
	input = strings.TrimSpace(input)

	// 2. Remove múltiplos espaços
	spaceRegex := regexp.MustCompile(`\s+`)
	input = spaceRegex.ReplaceAllString(input, " ")

	// 3. Remove caracteres de controle
	controlRegex := regexp.MustCompile(`[\x00-\x1F\x7F]`)
	input = controlRegex.ReplaceAllString(input, "")

	return input
}

// SanitizeMessage sanitiza uma mensagem do WhatsApp
// Remove saudações, emojis, pontuação excessiva e normaliza
func SanitizeMessage(msg string) string {
	if msg == "" {
		return ""
	}

	// 1. Sanitização básica
	msg = SanitizeInput(msg)

	// 2. Remove emojis (opcional - pode ser desabilitado)
	msg = removeEmojis(msg)

	// 3. Remove saudações comuns
	msg = removeSaudacoes(msg)

	// 4. Remove artigos e pronomes
	msg = removeArtigosPronomes(msg)

	// 5. Remove conectivos
	msg = removeConectivos(msg)

	// 6. Normaliza abreviações
	msg = normalizeAbreviacoes(msg)

	// 7. Remove pontuação excessiva
	msg = removePontuacaoExcessiva(msg)

	// 8. Limita tamanho máximo (proteção contra abuso)
	if len(msg) > 2000 {
		msg = msg[:2000]
	}

	return strings.TrimSpace(msg)
}

// SanitizeMessageLight sanitiza mensagem mantendo mais contexto
// Versão mais leve para quando precisamos de mais informações
func SanitizeMessageLight(msg string) string {
	if msg == "" {
		return ""
	}

	msg = SanitizeInput(msg)
	msg = removeEmojis(msg)
	msg = removePontuacaoExcessiva(msg)

	if len(msg) > 2000 {
		msg = msg[:2000]
	}

	return strings.TrimSpace(msg)
}
