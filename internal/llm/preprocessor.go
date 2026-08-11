// internal/llm/preprocessor.go
package llm

import (
	"regexp"
	"strings"
)

// PreprocessedMessage armazena a mensagem pré-processada
type PreprocessedMessage struct {
	Original string   // Mensagem original
	Cleaned  string   // Mensagem limpa para LLM
	Medidas  []string // Medidas extraídas
	Numeros  []string // Números encontrados
}

// Preprocessor realiza o pré-processamento de mensagens
type Preprocessor struct {
	saudacoes    []string
	pronomes     []string
	conectivos   []string
	medidasRegex []*regexp.Regexp
}

// NewPreprocessor cria um novo preprocessador
func NewPreprocessor() *Preprocessor {
	return &Preprocessor{
		saudacoes:  getSaudacoes(),
		pronomes:   getPronomesArtigos(),
		conectivos: getConectivos(),
		medidasRegex: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\d+[.,]?\d*\s*(ml|ML|litro|L|kg|g|cm|m|unidade|un|mg|mm|km)`),
			regexp.MustCompile(`(?i)\d+[.,]?\d*\s*(m[li]?|g|cm)`),
		},
	}
}

// Process processa a mensagem completa
func (p *Preprocessor) Process(mensagem string) *PreprocessedMessage {
	// Salva dados originais
	original := mensagem

	// Extrai medidas antes de remover
	medidas := p.extrairMedidas(mensagem)

	// Extrai números
	numeros := p.extrairNumeros(mensagem)

	// Limpa a mensagem
	cleaned := mensagem
	cleaned = p.removeSaudacoes(cleaned)
	cleaned = p.removePronomesArtigos(cleaned)
	cleaned = p.removeConectivos(cleaned)
	cleaned = p.removerMedidas(cleaned)
	cleaned = p.removePontuacaoExcessiva(cleaned)
	cleaned = p.normalizeEspacos(cleaned)
	cleaned = p.removeEmojis(cleaned)
	cleaned = p.normalizeAbreviacoes(cleaned)

	return &PreprocessedMessage{
		Original: original,
		Cleaned:  strings.TrimSpace(cleaned),
		Medidas:  medidas,
		Numeros:  numeros,
	}
}

// ============================================
// FUNÇÕES DE REMOÇÃO
// ============================================

// removeSaudacoes remove saudações do início da mensagem
func (p *Preprocessor) removeSaudacoes(mensagem string) string {
	for _, saudacao := range p.saudacoes {
		padrao := regexp.MustCompile(`(?i)^\s*` + regexp.QuoteMeta(saudacao) + `\s*[,!.]?\s*`)
		mensagem = padrao.ReplaceAllString(mensagem, "")
	}
	return mensagem
}

// removePronomesArtigos remove pronomes e artigos
func (p *Preprocessor) removePronomesArtigos(mensagem string) string {
	palavras := strings.Fields(mensagem)
	resultado := []string{}

	for _, palavra := range palavras {
		palavraLower := strings.ToLower(palavra)
		if !p.isStopWord(palavraLower, p.pronomes) {
			resultado = append(resultado, palavra)
		}
	}

	return strings.Join(resultado, " ")
}

// removeConectivos remove conectivos
func (p *Preprocessor) removeConectivos(mensagem string) string {
	palavras := strings.Fields(mensagem)
	resultado := []string{}

	for _, palavra := range palavras {
		palavraLower := strings.ToLower(palavra)
		if !p.isStopWord(palavraLower, p.conectivos) {
			resultado = append(resultado, palavra)
		}
	}

	return strings.Join(resultado, " ")
}

// removerMedidas remove medidas da mensagem
func (p *Preprocessor) removerMedidas(mensagem string) string {
	for _, re := range p.medidasRegex {
		mensagem = re.ReplaceAllString(mensagem, "")
	}
	return mensagem
}

// removePontuacaoExcessiva remove pontuação excessiva
func (p *Preprocessor) removePontuacaoExcessiva(mensagem string) string {
	// Remove múltiplos pontos de exclamação/interrogação
	re := regexp.MustCompile(`[!?]{2,}`)
	mensagem = re.ReplaceAllString(mensagem, " ")

	// Remove vírgulas e pontos no meio das palavras
	re = regexp.MustCompile(`[.,;:]`)
	mensagem = re.ReplaceAllString(mensagem, " ")

	return mensagem
}

// removeEmojis remove emojis da mensagem
func (p *Preprocessor) removeEmojis(mensagem string) string {
	// Remove emojis comuns
	re := regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{1F700}-\x{1F77F}\x{1F780}-\x{1F7FF}\x{1F800}-\x{1F8FF}\x{1F900}-\x{1F9FF}\x{1FA00}-\x{1FA6F}\x{1FA70}-\x{1FAFF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}]`)
	return re.ReplaceAllString(mensagem, "")
}

// ============================================
// FUNÇÕES DE EXTRAÇÃO
// ============================================

// extrairMedidas extrai medidas da mensagem
func (p *Preprocessor) extrairMedidas(mensagem string) []string {
	var encontradas []string
	seen := make(map[string]bool)

	for _, re := range p.medidasRegex {
		matches := re.FindAllString(mensagem, -1)
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				encontradas = append(encontradas, strings.TrimSpace(m))
			}
		}
	}

	return encontradas
}

// extrairNumeros extrai números da mensagem
func (p *Preprocessor) extrairNumeros(mensagem string) []string {
	var encontrados []string
	seen := make(map[string]bool)

	// Busca números (incluindo decimais)
	re := regexp.MustCompile(`\d+[.,]?\d*`)
	matches := re.FindAllString(mensagem, -1)

	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			encontrados = append(encontrados, m)
		}
	}

	return encontrados
}

// ============================================
// FUNÇÕES DE NORMALIZAÇÃO
// ============================================

// normalizeEspacos normaliza espaços
func (p *Preprocessor) normalizeEspacos(mensagem string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(mensagem, " ")
}

// normalizeAbreviacoes normaliza abreviações comuns
func (p *Preprocessor) normalizeAbreviacoes(mensagem string) string {
	abreviacoes := map[string]string{
		"vc":   "você",
		"vcê":  "você",
		"vcs":  "vocês",
		"c":    "com",
		"pq":   "porque",
		"tb":   "também",
		"tbm":  "também",
		"ñ":    "não",
		"ngm":  "ninguém",
		"td":   "tudo",
		"tds":  "todos",
		"vlw":  "valeu",
		"flw":  "falou",
		"blz":  "beleza",
		"obg":  "obrigado",
		"obgd": "obrigado",
		"q":    "que",
		"kd":   "cadê",
		"eh":   "é",
		"ta":   "está",
		"tá":   "está",
		"to":   "estou",
		"tô":   "estou",
	}

	for abrev, normal := range abreviacoes {
		// Substitui a abreviação com limites de palavra
		padrao := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(abrev) + `\b`)
		mensagem = padrao.ReplaceAllString(mensagem, normal)
	}

	return mensagem
}

// ============================================
// FUNÇÕES AUXILIARES
// ============================================

// isStopWord verifica se uma palavra é stop word
func (p *Preprocessor) isStopWord(palavra string, stopWords []string) bool {
	for _, sw := range stopWords {
		if palavra == sw {
			return true
		}
	}
	return false
}

// ============================================
// LISTAS DE PALAVRAS
// ============================================

func getSaudacoes() []string {
	return []string{
		"oi", "olá", "ola", "bom dia", "boa tarde", "boa noite",
		"tudo bem", "tudo bom", "oi tudo bem", "olá tudo bem",
		"bom", "boa", "beleza", "e aí", "e ai", "fala",
		"opa", "eae", "iae", "salve", "fala ai", "fala aí",
		"ei", "hey", "hello", "hi",
	}
}

func getPronomesArtigos() []string {
	return []string{
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
}

func getConectivos() []string {
	return []string{
		"e", "ou", "mas", "porém", "contudo", "todavia", "entretanto",
		"portanto", "logo", "assim", "pois", "porque", "porquê",
		"que", "como", "quando", "onde", "qual", "quanto",
		"por", "para", "com", "sem", "sobre", "sob", "entre",
	}
}
