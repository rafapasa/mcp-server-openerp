package llm

import (
	"regexp"
	"strings"
)

// PreprocessedMessage - simplificado pra LLM
type PreprocessedMessage struct {
	Original string
	Cleaned  string
}

type Preprocessor struct{}

func NewPreprocessor() *Preprocessor {
	return &Preprocessor{}
}

func (p *Preprocessor) Process(mensagem string) *PreprocessedMessage {
	original := mensagem
	cleaned := mensagem

	cleaned = p.normalizeEspacos(cleaned)
	cleaned = p.removeEmojisExcesso(cleaned)
	cleaned = p.normalizeAbreviacoes(cleaned)
	cleaned = strings.TrimSpace(cleaned)

	// se ficou vazia, devolve original
	if cleaned == "" {
		cleaned = strings.TrimSpace(original)
	}

	return &PreprocessedMessage{
		Original: original,
		Cleaned:  cleaned,
	}
}

func (p *Preprocessor) normalizeEspacos(s string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

// remove só excesso de emoji, mantém 1 se tiver significado (ex: 🔥)
func (p *Preprocessor) removeEmojisExcesso(s string) string {
	// remove sequencia de 3+ emojis iguais ou mistura exagerada, mas não todos
	// simples: limita para no max 2 emojis seguidos e tira o resto
	re := regexp.MustCompile(`([\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}]{3,})`)
	// por enquanto só limpa excesso de ! e ?
	re2 := regexp.MustCompile(`[!?]{3,}`)
	s = re2.ReplaceAllString(s, " ")
	_ = re
	return s
}

func (p *Preprocessor) normalizeAbreviacoes(s string) string {
	// só as que realmente mudam entendimento da LLM
	abreviacoes := map[string]string{
		"vc":  "você",
		"vcs": "vocês",
		"pq":  "porque",
		"tb":  "também",
		"tbm": "também",
		"obg": "obrigado",
		"blz": "beleza",
		"kd":  "cadê",
	}

	// faz replace com word boundary, case-insensitive
	lower := strings.ToLower(s)
	for abrev, normal := range abreviacoes {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(abrev) + `\b`)
		lower = re.ReplaceAllString(lower, normal)
	}
	// devolve lower normalizado? Não, LLM funciona melhor com lower mesmo
	// mas mantém original case pra observacao
	// aqui vamos devolver lower trimmed pra keywords
	return lower
}
