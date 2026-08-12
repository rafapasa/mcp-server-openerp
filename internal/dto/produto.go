// internal/dto/produto.go
package dto

import "time"

// ProdutoItem representa um item do cardápio para uso externo (DTO)
type ProdutoItem struct {
	ID           uint     `json:"id"`
	Nome         string   `json:"nome"`
	Categoria    string   `json:"categoria"`
	Descricao    string   `json:"descricao"`
	Preco        float64  `json:"preco"`
	Ingredientes []string `json:"ingredientes"`
	Disponivel   bool     `json:"disponivel"`
}

// ProdutoFiltro representa filtros para busca de produtos
type ProdutoFiltro struct {
	TenantID          string `json:"tenant_id"`
	Nome              string `json:"nome,omitempty"`
	Categoria         string `json:"categoria,omitempty"`
	ApenasDisponiveis bool   `json:"apenas_disponiveis,omitempty"`
	Limit             int    `json:"limit,omitempty"`
}

type ProdutoDTO struct {
	ID            uint      `json:"id"`
	TenantID      uint      `json:"tenant_id"`
	CategoriaID   *uint     `json:"categoria_id"`
	CategoriaNome string    `json:"categoria_nome"`
	Nome          string    `json:"nome"`
	Descricao     string    `json:"descricao"`
	Preco         float64   `json:"preco"`
	Disponivel    bool      `json:"disponivel"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
