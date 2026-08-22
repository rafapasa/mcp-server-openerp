package dto

type IntencaoCliente struct {
	Acao     string            `json:"acao"`
	Itens    []ItemPedidoInput `json:"itens,omitempty"`
	Mensagem string            `json:"mensagem,omitempty"`
}
