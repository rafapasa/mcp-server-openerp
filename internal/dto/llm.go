package dto

type IntencaoCliente struct {
	Acao     string         `json:"acao"`
	Itens    []ItemCarrinho `json:"itens,omitempty"`
	Mensagem string         `json:"mensagem,omitempty"`
}
