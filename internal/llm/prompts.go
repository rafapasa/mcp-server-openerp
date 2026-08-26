package llm

const (
	PromptSystemBase = `Você é atendente do %s via WhatsApp.`

	PromptExtractKeywords = `
Você é extrator de pedidos. TEXTO: "%s"
Retorne APENAS JSON: {"keywords":[{"nome": "x bacon", "qtd":2},{"nome": "coca cola", "qtd":1}]}
Sem markdown, sem texto extra.
`

	PromptResolverIDs = `
Cardápio (id - nome):
%s
Keywords e suas quantidades: %s
Retorne APENAS JSON: {"itens":[{"id":12,"qtd":1,"obs":""}]}
Lista completa para validação:
%s
`

	PromptTranscribeSimple = `Transcreva o áudio em PT-BR fielmente. Retorne apenas a transcrição limpa.`
	PromptVisionDescribe   = `Descreva o pedido do cliente na imagem de forma objetiva. Foco em produtos e quantidades.`

	PromptErroGenerico = `Desculpe, problema temporário. Pode repetir?`
)
