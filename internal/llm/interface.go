// internal/llm/interface.go
package llm

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

// LLMClient é a interface que todos os provedores de LLM devem implementar
type LLMClient interface {
	Generate(prompt string) (string, error)
	GenerateWithContext(ctx context.Context, prompt string) (string, error)
	GetModel() string
	GetProvider() string
	ExtractIntent(mensagem string, cardapio []dto.ProdutoItem) (*IntencaoCliente, error)
	CorrigirNomes(nomesNaoEncontrados []string, produtosEncontrados map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error)
}

// IntencaoCliente representa a intenção extraída da mensagem
type IntencaoCliente struct {
	Acao     string                `json:"acao"`
	Itens    []dto.ItemPedidoInput `json:"itens,omitempty"`
	Mensagem string                `json:"mensagem,omitempty"`
}
