package intent

import (
	"regexp"
	"strings"
)

type Intent string

const (
	IntentGreeting  Intent = "greeting"   // bom dia, ola, oi
	IntentSmallTalk Intent = "small_talk" // tudo bem, obrigado, valeu
	IntentAdd       Intent = "add"        // adiciona, quero, coloca
	IntentDel       Intent = "del"        // remove, tira
	IntentUpdate    Intent = "update"     // troca, muda
	IntentView      Intent = "view"       // ver carrinho, quanto deu
	IntentCheckout  Intent = "checkout"   // finalizar, fechar pedido
	IntentOther     Intent = "other"      // cai pro LLM caro
)

var (
	reGreeting  = regexp.MustCompile(`(?i)^(oi|olá|ola|bom dia|boa tarde|boa noite|e ai|fala|hello|hey)(\s|$|!|,)`)
	reSmallTalk = regexp.MustCompile(`(?i)(tudo bem|como vai|obrigado|obrigada|valeu|tchau|até|blz|beleza|ok$)`)
	reAdd       = regexp.MustCompile(`(?i)(adiciona|add|coloca|quero|manda|pede|\+)`)
	reDel       = regexp.MustCompile(`(?i)(remove|tira|exclui|cancela|-\s)`)
	reCheckout  = regexp.MustCompile(`(?i)(finaliza|fechar pedido|fecha|checkout|enviar pedido)`)
	reView      = regexp.MustCompile(`(?i)(ver carrinho|meu carrinho|quanto deu|total)`)
)

// Classifier rápido - roda ANTES do LLM, economiza 80% de tokens
func Classify(text string) Intent {
	t := strings.TrimSpace(strings.ToLower(text))
	if t == "" {
		return IntentOther
	}
	// 1. Saudação pura - resposta instantânea, sem LLM
	if reGreeting.MatchString(t) && len(t) < 25 {
		return IntentGreeting
	}
	if reSmallTalk.MatchString(t) && len(t) < 30 {
		return IntentSmallTalk
	}
	// 2. Ações do carrinho
	if reCheckout.MatchString(t) {
		return IntentCheckout
	}
	if reView.MatchString(t) {
		return IntentView
	}
	if reDel.MatchString(t) {
		return IntentDel
	}
	if reAdd.MatchString(t) {
		return IntentAdd
	}
	// 3. Só se não for nada disso, chama LLM
	return IntentOther
}

// Respostas humanas pra saudação - sem precisar de DeepSeek/Gemini
func GreetingResponse(nome string, hour int) string {
	saudacao := "Olá"
	if hour < 12 {
		saudacao = "Bom dia"
	} else if hour < 18 {
		saudacao = "Boa tarde"
	} else {
		saudacao = "Boa noite"
	}
	if nome == "" {
		nome = "por aqui"
	}
	// Respostas variadas pra não parecer robô
	return saudacao + " " + nome + "! Tudo bem? 😊\nComo posso te ajudar hoje? Posso mostrar o cardápio ou ver seu carrinho."
}

func SmallTalkResponse(text string) string {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "obrigado"), strings.Contains(t, "valeu"):
		return "Imagina! Sempre à disposição 🙏 Precisa de mais alguma coisa?"
	case strings.Contains(t, "tudo bem"):
		return "Tudo ótimo por aqui! E por aí? Bora fazer um pedido?"
	default:
		return "Haha, tamo junto! Quer dar uma olhada no cardápio?"
	}
}
