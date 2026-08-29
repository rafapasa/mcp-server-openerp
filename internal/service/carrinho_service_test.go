package service

import (
	"testing"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/stretchr/testify/assert"
)

func TestCarrinhoService_CalcularTotal(t *testing.T) {
	tests := []struct {
		name     string
		itens    []dto.ItemCarrinho
		expected float64
	}{
		{
			name:     "carrinho vazio",
			itens:    []dto.ItemCarrinho{},
			expected: 0.0,
		},
		{
			name: "um item",
			itens: []dto.ItemCarrinho{
				{Preco: 10.0, Quantidade: 2},
			},
			expected: 20.0,
		},
		{
			name: "vários itens",
			itens: []dto.ItemCarrinho{
				{Preco: 5.0, Quantidade: 3},
				{Preco: 7.5, Quantidade: 2},
			},
			expected: 15.0 + 15.0, // 3*5=15, 2*7.5=15
		},
	}

	service := &CarrinhoService{} // não precisa de dependências para este teste

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carrinho := &dto.Carrinho{Itens: tt.itens}
			total := service.CalcularTotal(carrinho)
			assert.Equal(t, tt.expected, total)
		})
	}
}
