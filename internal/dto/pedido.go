// internal/dto/pedido.go
package dto

// ItemPedidoInput representa um item do pedido (entrada)
type ItemPedidoInput struct {
	ProdutoItem   ProdutoItem `json:"produto_item"`
	Quantidade    int         `json:"quantidade"`
	Observacao    string      `json:"observacao"`
	PrecoUnitario float64     `json:"preco_unitario,omitempty"`
}

// PedidoExtraido representa o pedido extraído pela IA
type PedidoExtraido struct {
	Itens       []ItemPedidoInput `json:"itens"`
	Bebidas     []ItemPedidoInput `json:"bebidas,omitempty"`
	Observacoes string            `json:"observacoes,omitempty"`
}

// PedidoConfirmado representa o pedido após processamento
type PedidoConfirmado struct {
	ID            int               `json:"id"`
	TenantID      string            `json:"tenant_id"`
	ClienteID     string            `json:"cliente_id"`
	ClienteNome   string            `json:"cliente_nome"`
	Itens         []ItemPedidoInput `json:"itens"`
	Total         float64           `json:"total"`
	TempoEstimado int               `json:"tempo_estimado"`
	Status        string            `json:"status"`
	CriadoEm      string            `json:"criado_em"`
}
