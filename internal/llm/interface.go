package llm

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

type LLMClient interface {
	GetProvider() string
	GetModel() string
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error)
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

// LLMProvider define a interface do provedor LLM externo (OpenAI, Gemini, etc.)
// Todas as chamadas para modelos de linguagem, áudio e visão passam por aqui.
type LLMProvider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error)
	DescribeImage(ctx context.Context, image []byte, caption string) (string, error)
	ResolveItemsByMenu(ctx context.Context, input dto.MessageInput, cardapio []dto.ProdutoItem) (*dto.IntencaoCliente, error)
	ClassificarEExtrairKeywords(ctx context.Context, texto, contextoCarrinho string) (*IntencaoEKeywordsResult, error)
	ResolverItensByKeyWords(ctx context.Context, keywords []LLMKeywordItemResult, cardapioReduzido []dto.ProdutoItem) ([]dto.ItemCarrinho, error)
	Higienizar(texto string) string
	GetProvider() string
}
