package llm

const (
	PromptSystemBase = `Você é atendente do %s via WhatsApp.`

	// PromptContextoLoja injeta o contexto multi-tenant (nome + segmento) no prompt do LLM.
	PromptContextoLoja = `LOJA: %s
SEGMENTO: %s`

	PromptTranscribeSimple = `Transcreva o áudio em PT-BR fielmente. Retorne apenas a transcrição limpa.`
	PromptVisionDescribe   = `Descreva o pedido do cliente na imagem de forma objetiva. Foco em produtos e quantidades.`
	PromptErroGenerico     = `Desculpe, problema temporário. Pode repetir?`

	PromptClassificarEExtrairKeywords = `
Você é classificador de intenção de cliente de delivery/comércio.

CONTEXTO DA LOJA:
%s

TEXTO DO CLIENTE: "%s"
CONTEXTO DO CARRINHO ATUAL: %s

Classifique em UMA ação e extraia keywords de produtos (se houver).

AÇÕES POSSÍVEIS:
- adicionar: quer colocar itens no carrinho
- remover: quer tirar itens
- finalizar: quer fechar o pedido
- limpar: quer esvaziar o carrinho
- visualizar: quer ver o que tem no carrinho
- listar_categorias: pergunta quais categorias/tipos de produtos existem
- listar_produtos: quer ver produtos (pode ter filtro)
- conversa: saudação, agradecimento, dúvida geral sem pedido

REGRAS:
- Se for conversa, preencha "resposta" com uma frase cordial e curta.
- Se NÃO for conversa, deixe "resposta" vazia.
- Em "keywords" coloque apenas palavras que parecem produtos + quantidade + unidade de medida (se houver).
- Exemplos de unidade: 600ml, lata, kg, un, com ovo, grande, etc.
- Se não houver quantidade explícita, use 1.
- Em "filtro" coloque o filtro quando a ação for listar_produtos (ex: "lanche", "refri").

Retorne APENAS JSON válido:
{
  "acao": "adicionar|remover|finalizar|limpar|visualizar|listar_categorias|listar_produtos|conversa",
  "resposta": "",
  "filtro": "",
  "keywords": [
    {"nome": "coca", "qtd": 2, "unidade": "lata"}
  ]
}
Sem markdown, sem texto extra.
`

	PromptResolverByKeywords = `
Você é um resolvedor de produtos de cardápio.

O cliente pediu estas keywords:
%s

CARDÁPIO DISPONÍVEL (já filtrado):
%s

Para cada keyword, escolha o ID do produto que melhor corresponde.
Se não tiver certeza, não invente ID.
Quantidade deve ser a mesma da keyword (ou 1 se não informada).
Unidade ajuda a desambiguar (ex: coca 600ml vs coca lata).

Retorne APENAS JSON válido:
{
  "itens": [
    {"id": 12, "qtd": 2, "obs": ""}
  ]
}
Sem markdown.
`

	PromptResolveByMenu = `
Você é atendente de delivery. Analise a mensagem do cliente e o cardápio.

CONTEXTO DA LOJA:
%s

MENSAGEM DO CLIENTE: "%s"

CARDÁPIO (id - nome - preço):
%s

Tarefas:
1. Classifique a intenção: adicionar | remover | finalizar | limpar | visualizar | listar_categorias | listar_produtos | conversa
2. Se for conversa, preencha "resposta" com frase cordial e curta.
3. Se for adicionar/remover, resolva os IDs exatos do cardápio e quantidades.
4. Em "filtro" coloque o filtro quando a ação for listar_produtos.

Retorne APENAS JSON válido:
{
  "acao": "...",
  "resposta": "",
  "filtro": "",
  "itens": [
    {"id": 12, "qtd": 1, "obs": ""}
  ]
}
Sem markdown.
`
)
