package llm

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

type LLMClient interface {
	GetProvider() string
	GetModel() string
	ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*dto.IntencaoCliente, error)
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	TranscribeAudio(ctx context.Context, audio []byte) (string, error)
	DescribeImage(ctx context.Context, image []byte, prompt string) (string, error)
}

type TextLLM interface {
	LLMClient
}

type AudioLLM interface {
	LLMClient
}

type VisionLLM interface {
	LLMClient
}
