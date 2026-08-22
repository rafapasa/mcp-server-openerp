package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
)

func newLLMClient(provider string, cfg *config.Config) (LLMClient, error) {
	switch provider {
	case "openai":
		return NewOpenAILLM(cfg), nil
	case "groq":
		return NewGroqLLM(cfg), nil
	case "gemini":
		return NewGeminiLLM(cfg), nil
	case "deepseek":
		return NewDeepSeekLLM(cfg), nil
	default:
		return nil, fmt.Errorf("provedor não suportado: %s", provider)
	}
}

func NewLLMText(cfg *config.Config) TextLLM {
	provider := strings.ToLower(cfg.LlmText)
	logger.LogInfo(fmt.Sprintf("[LLM_TEXT] Provider: %s", provider))
	llm, err := newLLMClient(provider, cfg)
	if err != nil {
		logger.Error(context.Background(), fmt.Sprintf("Erro ao criar llm de texto [%s]: %v", provider, err.Error()))
		return nil
	}
	return llm
}

func NewLLMAudio(cfg *config.Config) AudioLLM {
	provider := strings.ToLower(cfg.LlmAudio)
	logger.LogInfo(fmt.Sprintf("[LLM_AUDIO] Provider: %s", provider))
	llm, err := newLLMClient(provider, cfg)
	if err != nil {
		logger.Error(context.Background(), fmt.Sprintf("Erro ao criar llm de audio [%s]: %v", provider, err.Error()))
		return nil
	}
	return llm
}

func NewLLMVision(cfg *config.Config) VisionLLM {
	provider := strings.ToLower(cfg.LlmVision)
	logger.LogInfo(fmt.Sprintf("[LLM_VISION] Provider: %s", provider))
	llm, err := newLLMClient(provider, cfg)
	if err != nil {
		logger.Error(context.Background(), fmt.Sprintf("Erro ao criar llm de vision [%s]: %v", provider, err.Error()))
		return nil
	}
	return llm
}
