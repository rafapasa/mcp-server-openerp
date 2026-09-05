// internal/intent/classifier.go - FINAL V2 - sem SmallTalkResponse quebrada
package intent

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// IntentType identifica a intenção reconhecida em uma mensagem.
type IntentType int

const (
	// IntentNone indica que nenhuma intenção foi reconhecida.
	IntentNone IntentType = iota
	// IntentGreeting indica uma saudação.
	IntentGreeting
	// IntentGreetingWithAdd indica saudação acompanhada de um item.
	IntentGreetingWithAdd
	// IntentSmallTalk indica conversa informal.
	IntentSmallTalk
	// IntentThanks indica agradecimento.
	IntentThanks
	// IntentViewCart indica consulta do carrinho.
	IntentViewCart
	// IntentClearCart indica limpeza do carrinho.
	IntentClearCart
	// IntentAdd indica adição de item.
	IntentAdd
	// IntentRemove indica remoção de item.
	IntentRemove
	// IntentCheckout indica finalização do pedido.
	IntentCheckout
	// IntentFalarComAtendente indica solicitação de atendimento humano.
	IntentFalarComAtendente
	// IntentVoltarProBot indica retorno ao atendimento automatizado.
	IntentVoltarProBot
	// IntentOther indica uma intenção não classificada.
	IntentOther
)

// Result contém a intenção reconhecida, o texto restante e sua confiança.
type Result struct {
	Type      IntentType
	CleanRest string
	Score     float64
}

var (
	reMultiSpace  = regexp.MustCompile(`\s+`)
	greetingsBase = []string{"bom dia", "boa tarde", "boa noite", "boa madrugada", "ola", "oi", "opa", "e ai", "eae", "fala", "salve"}
	thanksBase    = []string{"obrigado", "obrigada", "valeu", "vlw", "tmj", "tamo junto", "grato", "obg"}
	smallTalkBase = []string{"tudo bem", "td bem", "como vai", "beleza", "suave", "tranquilo", "tudo bom", "td bom"}
	viewBase      = []string{"ver carrinho", "meu carrinho", "ver pedido", "o que tenho", "q tenho", "carrinho"}
	limparBase    = []string{"limpar carrinho", "limpar tudo", "limpar pedido", "limpe tudo", "esvaziar carrinho", "apagar carrinho", "zera o carrinho"}
)

func IsFalarComAtendente(raw string) bool {
	norm := normalize(raw)
	return strings.Contains(norm, "atendente") ||
		strings.Contains(norm, "falar com humano") ||
		strings.Contains(norm, "suporte") ||
		strings.Contains(norm, "ajuda")
}

// IsVoltarProBot informa se a mensagem solicita o retorno ao bot.
func IsVoltarProBot(raw string) bool {
	norm := normalize(raw)
	return strings.Contains(norm, "voltar pro bot") ||
		strings.Contains(norm, "voltar para o bot") ||
		strings.Contains(norm, "sair do atendimento")
}

func normalize(s string) string {
	s = strings.ToLower(s)

	// Remove acentos
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ = transform.String(t, s)

	// FIX: Go não suporta \1 - colapsa!!!???... manualmente
	s = collapseRepeatedPunct(s)

	s = reMultiSpace.ReplaceAllString(strings.TrimSpace(s), " ")
	return s
}

// colapsa!!! ->!,??? ->?,... ->.
func collapseRepeatedPunct(s string) string {
	var b strings.Builder
	var prev rune
	var count int

	for _, r := range s {
		if r == '!' || r == '?' || r == '.' {
			if r == prev {
				count++
				if count == 1 { // já escreveu um, ignora repetidos
					continue
				}
			} else {
				count = 0
				b.WriteRune(r)
			}
			prev = r
		} else {
			b.WriteRune(r)
			prev = 0
			count = 0
		}
	}
	return b.String()
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			d[i][j] = min3(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
		}
	}
	return d[la][lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func metaphonePT(s string) string {
	s = normalize(s)
	r := strings.NewReplacer(
		"ph", "f", "ç", "s", "ss", "s", "sc", "s", "sç", "s",
		"lh", "l", "nh", "n", "ão", "am", "ã", "a", "õ", "o",
		"w", "v", "y", "i", "  ", " ",
	)
	return r.Replace(s)
}

func similarity(a, b string) float64 {
	aN, bN := normalize(a), normalize(b)
	if aN == bN {
		return 1.0
	}
	if strings.Contains(aN, bN) || strings.Contains(bN, aN) {
		return 0.9
	}
	if metaphonePT(aN) == metaphonePT(bN) {
		return 0.85
	}
	maxLen := len(aN)
	if len(bN) > maxLen {
		maxLen = len(bN)
	}
	if maxLen == 0 {
		return 0
	}
	dist := levenshtein(aN, bN)
	return 1.0 - float64(dist)/float64(maxLen)
}

func findBestMatch(input string, dict []string) (string, float64) {
	best := ""
	bestScore := 0.0
	for _, w := range dict {
		s := similarity(input, w)
		if s > bestScore {
			bestScore = s
			best = w
		}
	}
	return best, bestScore
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ClassifyV2 classifica uma mensagem considerando a última saudação recebida.
func ClassifyV2(raw string, lastGreeting time.Time) Result {
	norm := normalize(raw)
	if norm == "" {
		return Result{Type: IntentNone}
	}
	if IsVoltarProBot(norm) {
		return Result{Type: IntentVoltarProBot, Score: 1}
	}
	if IsFalarComAtendente(norm) {
		return Result{Type: IntentFalarComAtendente, Score: 1}
	}
	// anti-spam saudação 3min
	if !lastGreeting.IsZero() && time.Since(lastGreeting) < 3*time.Minute {
		if len(norm) < 15 {
			_, score := findBestMatch(norm, greetingsBase)
			if score > 0.7 {
				return Result{Type: IntentNone}
			}
		}
	}
	// multi-intento
	for _, g := range greetingsBase {
		parts := strings.Fields(norm)
		if len(parts) == 0 {
			continue
		}
		prefix := strings.Join(parts[:minInt(2, len(parts))], " ")
		if similarity(prefix, g) > 0.65 || similarity(parts[0], g) > 0.65 {
			rest := strings.TrimSpace(strings.TrimPrefix(norm, prefix))
			if len(rest) > 2 {
				return Result{Type: IntentGreetingWithAdd, CleanRest: rest, Score: 0.85}
			}
			return Result{Type: IntentGreeting, Score: 0.85}
		}
	}
	if _, s := findBestMatch(norm, thanksBase); s > 0.75 {
		return Result{Type: IntentThanks, Score: s}
	}
	if _, s := findBestMatch(norm, smallTalkBase); s > 0.7 {
		return Result{Type: IntentSmallTalk, Score: s}
	}
	// limpar carrinho precisa vir antes do viewBase (que pega "carrinho" por contains)
	if _, s := findBestMatch(norm, limparBase); s > 0.75 {
		return Result{Type: IntentClearCart, Score: s}
	}
	if _, s := findBestMatch(norm, viewBase); s > 0.75 {
		return Result{Type: IntentViewCart, Score: s}
	}
	if strings.Contains(norm, "tudo bem") && len(norm) < 25 {
		return Result{Type: IntentGreeting, Score: 0.8}
	}
	return Result{Type: IntentOther, CleanRest: norm}
}

// COMPATIBILIDADE - para seu handlers.go não quebrar
// Classify classifica uma mensagem usando o classificador padrão.
func Classify(raw string) IntentType {
	res := ClassifyV2(raw, time.Time{})
	return res.Type
}

// GreetingResponse gera uma resposta apropriada para uma saudação.
func GreetingResponse(nome string, hour int) string {
	saudacao := "Olá"
	if hour >= 5 && hour < 12 {
		saudacao = "Bom dia"
	} else if hour < 18 {
		saudacao = "Boa tarde"
	} else {
		saudacao = "Boa noite"
	}
	if nome != "" {
		return saudacao + ", " + nome + "! 😊 O que vai querer hoje?"
	}
	return saudacao + "! 😊 O que vai querer hoje?"
}

// SmallTalkResponse gera uma resposta para uma mensagem informal.
func SmallTalkResponse(raw string) string {
	norm := normalize(raw)
	if strings.Contains(norm, "tudo bem") || strings.Contains(norm, "td bem") {
		return "Tudo ótimo por aqui! 😊 E com você? O que vai querer hoje?"
	}
	if strings.Contains(norm, "obrigado") || strings.Contains(norm, "valeu") || strings.Contains(norm, "obg") {
		return "Por nada! 😊 Precisa de mais alguma coisa?"
	}
	return "Haha, tudo certo! 😊 O que vai querer pedir hoje?"
}

// Para Thanks separado se quiser
// ThanksResponse gera uma resposta para um agradecimento.
func ThanksResponse() string {
	return "Por nada! 😊 Se precisar, estou por aqui."
}

// ViewCartResponse gera uma resposta para a consulta do carrinho.
func ViewCartResponse() string {
	return "Claro! Deixa eu ver seu carrinho..."
}
