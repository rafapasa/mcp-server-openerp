// internal/dto/pedido_dto.go
package dto

import "time"

type PedidoDTO struct {
	ID                uint            `json:"id"`
	TenantID          uint            `json:"tenant_id"`
	ClienteID         *uint           `json:"cliente_id"`
	ClienteNome       string          `json:"cliente_nome"`
	ClienteTelefone   string          `json:"cliente_telefone"`
	EnderecoEntregaID *uint           `json:"endereco_entrega_id"`
	EnderecoEntrega   *EnderecoDTO    `json:"endereco_entrega,omitempty"`
	Itens             []ItemPedidoDTO `json:"itens"`
	Total             float64         `json:"total"`
	Status            string          `json:"status"`
	Observacoes       string          `json:"observacoes"`
	TempoEstimado     int             `json:"tempo_estimado"`
	Origem            string          `json:"origem"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type ItemPedidoDTO struct {
	Nome       string  `json:"nome"`
	Quantidade int     `json:"quantidade"`
	Preco      float64 `json:"preco"`
	Observacao string  `json:"observacao"`
}

type CriarPedidoRequest struct {
	TenantID          uint            `json:"tenant_id"`
	ClienteID         *uint           `json:"cliente_id"`
	EnderecoEntregaID *uint           `json:"endereco_entrega_id"`
	Itens             []ItemPedidoDTO `json:"itens"`
	Observacoes       string          `json:"observacoes"`
	Origem            string          `json:"origem"`
}

// Total retorna o valor total do pedido
func (r *CriarPedidoRequest) Total() float32 {
	var total float64
	for _, item := range r.Itens {
		total += float64(item.Quantidade) * item.Preco
	}
	return float32(total)
}

// Se quiser em float64 (recomendado pro seu Pedido.Total que é float64):
func (r *CriarPedidoRequest) TotalFloat64() float64 {
	var total float64
	for _, item := range r.Itens {
		total += float64(item.Quantidade) * item.Preco
	}
	return total
}

type PedidoListResponseDTO struct {
	Pedidos    []PedidoDTO `json:"pedidos"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int64       `json:"total_pages"`
}
