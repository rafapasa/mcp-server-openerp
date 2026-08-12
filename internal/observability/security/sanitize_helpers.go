// internal/observability/security/sanitize_helpers.go
package security

import (
	"regexp"
	"strings"
)

// ============================================
// REMOÇÃO DE EMOJIS
// ============================================

// removeEmojis remove emojis da mensagem
func removeEmojis(msg string) string {
	// Emojis comuns (U+1F600 - U+1F64F)
	emojiRegex := regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{1F700}-\x{1F77F}\x{1F780}-\x{1F7FF}\x{1F800}-\x{1F8FF}\x{1F900}-\x{1F9FF}\x{1FA00}-\x{1FA6F}\x{1FA70}-\x{1FAFF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}]`)
	return emojiRegex.ReplaceAllString(msg, "")
}

// ============================================
// REMOÇÃO DE SAUDAÇÕES
// ============================================

var saudacoes = []string{
	"oi", "olá", "ola", "bom dia", "boa tarde", "boa noite",
	"tudo bem", "tudo bom", "oi tudo bem", "olá tudo bem",
	"bom", "boa", "beleza", "e aí", "e ai", "fala",
	"opa", "eae", "iae", "salve", "fala ai", "fala aí",
	"ei", "hey", "hello", "hi", "blz", "beleza",
}

func removeSaudacoes(msg string) string {
	for _, saudacao := range saudacoes {
		padrao := regexp.MustCompile(`(?i)^\s*` + regexp.QuoteMeta(saudacao) + `\s*[,!.]?\s*`)
		msg = padrao.ReplaceAllString(msg, "")
	}
	return msg
}

// ============================================
// REMOÇÃO DE ARTIGOS E PRONOMES
// ============================================

var artigosPronomes = []string{
	"eu", "tu", "ele", "ela", "nós", "vós", "eles", "elas",
	"me", "te", "se", "nos", "vos",
	"o", "a", "os", "as",
	"um", "uma", "uns", "umas",
	"este", "esta", "estes", "estas",
	"esse", "essa", "esses", "essas",
	"aquele", "aquela", "aqueles", "aquelas",
	"meu", "minha", "meus", "minhas",
	"teu", "tua", "teus", "tuas",
	"seu", "sua", "seus", "suas",
	"nosso", "nossa", "nossos", "nossas",
}

func removeArtigosPronomes(msg string) string {
	palavras := strings.Fields(msg)
	resultado := []string{}

	for _, palavra := range palavras {
		palavraLower := strings.ToLower(palavra)
		if !isStopWord(palavraLower, artigosPronomes) {
			resultado = append(resultado, palavra)
		}
	}

	return strings.Join(resultado, " ")
}

// ============================================
// REMOÇÃO DE CONECTIVOS
// ============================================

var conectivos = []string{
	"e", "ou", "mas", "porém", "contudo", "todavia", "entretanto",
	"portanto", "logo", "assim", "pois", "porque", "porquê",
	"que", "como", "quando", "onde", "qual", "quanto",
	"por", "para", "com", "sem", "sobre", "sob", "entre",
}

func removeConectivos(msg string) string {
	palavras := strings.Fields(msg)
	resultado := []string{}

	for _, palavra := range palavras {
		palavraLower := strings.ToLower(palavra)
		if !isStopWord(palavraLower, conectivos) {
			resultado = append(resultado, palavra)
		}
	}

	return strings.Join(resultado, " ")
}

// ============================================
// NORMALIZAÇÃO DE ABREVIAÇÕES
// ============================================

var abreviacoes = map[string]string{
	"vc": "você", "vcê": "você", "vcs": "vocês",
	"c": "com", "pq": "porque", "tb": "também", "tbm": "também",
	"ñ": "não", "ngm": "ninguém", "td": "tudo", "tds": "todos",
	"vlw": "valeu", "flw": "falou", "blz": "beleza",
	"obg": "obrigado", "obgd": "obrigado",
	"q": "que", "kd": "cadê", "eh": "é", "ta": "está",
	"tá": "está", "to": "estou", "tô": "estou",
}

func normalizeAbreviacoes(msg string) string {
	for abrev, normal := range abreviacoes {
		padrao := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(abrev) + `\b`)
		msg = padrao.ReplaceAllString(msg, normal)
	}
	return msg
}

// ============================================
// REMOÇÃO DE PONTUAÇÃO EXCESSIVA
// ============================================

func removePontuacaoExcessiva(msg string) string {
	// Remove múltiplos pontos de exclamação/interrogação
	re := regexp.MustCompile(`[!?]{2,}`)
	msg = re.ReplaceAllString(msg, " ")

	// Remove vírgulas e pontos no meio das palavras
	re = regexp.MustCompile(`[.,;:]`)
	msg = re.ReplaceAllString(msg, " ")

	return msg
}

// ============================================
// FUNÇÕES AUXILIARES
// ============================================

func isStopWord(palavra string, stopWords []string) bool {
	for _, sw := range stopWords {
		if palavra == sw {
			return true
		}
	}
	return false
}
