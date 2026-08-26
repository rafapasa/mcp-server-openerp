package llm

import (
	"context"
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
