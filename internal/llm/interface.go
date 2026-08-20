// internal/llm/interface.go
package llm

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

type LLMClient interface {
	Generate(prompt string) (string, error)
	GenerateWithContext(ctx context.Context, prompt string) (string, error)
	// NOVO - vision (Gemini). Providers sem vision retornam erro ou fallback para Generate
	GenerateWithImage(ctx context.Context, prompt, b64Data, mimeType string) (string, error)
	GetModel() string
	GetProvider() string
	ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*IntencaoCliente, error)
	CorrigirNomes(ctx context.Context, nomesNaoEncontrados []string, produtosEncontrados map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error)
}

type IntencaoCliente struct {
	Acao     string                `json:"acao"`
	Itens    []dto.ItemPedidoInput `json:"itens,omitempty"`
	Mensagem string                `json:"mensagem,omitempty"`
}

// Opcional: interface pra quem tem audio
type Transcriber interface {
	Transcribe(ctx context.Context, audioBytes []byte) (string, error)
}
