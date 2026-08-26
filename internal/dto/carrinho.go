// internal/dto/carrinho.go
package dto

import "time"

// ItemCarrinho representa um item no carrinho
type ItemCarrinho struct {
	ProdutoItem ProdutoItem `json:"produto_item"`
	Quantidade  int         `json:"quantidade"`
	Observacao  string      `json:"observacao,omitempty"`
	Preco       float64     `json:"preco"`
}

// Carrinho representa o carrinho de um cliente
type Carrinho struct {
	ClienteID string         `json:"cliente_id"`
	TenantID  string         `json:"tenant_id"`
	Itens     []ItemCarrinho `json:"itens"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
