package dto

type IntencaoCliente struct {
	Acao     string         `json:"acao"`
	Resposta string         `json:"resposta,omitempty"` // preenchida quando acao == conversa
	Filtro   string         `json:"filtro,omitempty"`   // usado em listar_produtos
	Itens    []ItemCarrinho `json:"itens,omitempty"`
	Mensagem string         `json:"mensagem,omitempty"` // texto original do usuário
}
