package llm

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

type UnifiedLLM struct {
	textClient   TextLLM
	audioClient  AudioLLM
	visionClient VisionLLM
}

func NewUnifiedLLM(cfg *config.Config) *UnifiedLLM {
	// SEMPRE cria os 3 clients - cada um implementa TUDO
	// Se env não tem key, o construtor loga warning mas não quebra
	return &UnifiedLLM{
		textClient:   NewLLMText(cfg),   // deepseek implementa texto
		audioClient:  NewLLMAudio(cfg),  // groq implementa audio
		visionClient: NewLLMVision(cfg), // gemini implementa vision
	}
}

// Text - delega pro textClient
func (u *UnifiedLLM) ExtractIntent(ctx context.Context, msg string, cardapio []dto.ProdutoItem) (*dto.IntencaoCliente, error) {
	return u.textClient.ExtractIntent(ctx, msg, cardapio)
}
func (u *UnifiedLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	return u.textClient.GenerateResponse(ctx, prompt)
}

// Audio - delega pro audioClient
func (u *UnifiedLLM) TranscribeAudio(ctx context.Context, audio []byte) (string, error) {
	return u.audioClient.TranscribeAudio(ctx, audio)
}

// Vision - delega pro visionClient
func (u *UnifiedLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	return u.visionClient.DescribeImage(ctx, image, prompt)
}
func (u *UnifiedLLM) GenerateWithImage(ctx context.Context, prompt, b64, mime string) (string, error) {
	return u.DescribeImage(ctx, []byte(b64), prompt)
}
