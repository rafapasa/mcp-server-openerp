package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

func newLLMClient(provider string, cfg *config.Config) (LLMClient, error) {
	logger.LogInfo(fmt.Sprintf("[FACTORY] Tentando criar client provider=%s", provider))

	var client LLMClient
	var err error

	switch provider {
	case "openai":
		if cfg.LlmOpenAiApiKey == "" {
			logger.LogInfo("[FACTORY] OpenAI Key vazia, pulando")
			return nil, fmt.Errorf("openai key vazia")
		}
		client = NewOpenAILLM(cfg)
	case "groq":
		if cfg.LlmGroqApiKey == "" {
			logger.LogInfo("[FACTORY] Groq Key vazia, pulando")
			return nil, fmt.Errorf("groq key vazia")
		}
		client = NewGroqLLM(cfg)
	case "gemini":
		if cfg.LlmGeminiApiKey == "" {
			logger.LogInfo("[FACTORY] Gemini Key vazia, pulando")
			return nil, fmt.Errorf("gemini key vazia")
		}
		client = NewGeminiLLM(cfg)
	case "deepseek":
		if cfg.LlmDeepSeekApiKey == "" {
			logger.LogInfo("[FACTORY] DeepSeek Key vazia, pulando")
			return nil, fmt.Errorf("deepseek key vazia")
		}
		client = NewDeepSeekLLM(cfg)
	default:
		logger.LogInfo(fmt.Sprintf("[FACTORY] Provedor não suportado: %s", provider))
		return nil, fmt.Errorf("provedor não suportado: %s", provider)
	}

	// Log de sucesso
	logger.Info(
		context.Background(), "LLM client criado com sucesso",
		zap.String("provider", client.GetProvider()),
		zap.String("model", client.GetModel()),
	)

	return client, err
}

func NewLLMText(cfg *config.Config) TextLLM {
	provider := strings.ToLower(strings.TrimSpace(cfg.LlmText))
	if provider == "" {
		provider = "groq" // default rápido e barato pra keyword
	}

	logger.LogInfo(fmt.Sprintf("[FACTORY_TEXT] Provider configurado: %s", provider))

	llm, err := newLLMClient(provider, cfg)
	if err != nil {
		logger.Error(
			context.Background(), fmt.Sprintf("[FACTORY_TEXT] Falha ao criar client texto [%s]", provider),
			zap.Error(err),
		)
		// Fallback: tenta groq
		if provider != "groq" {
			logger.LogInfo("[FACTORY_TEXT] Tentando fallback para groq")
			llm, _ = newLLMClient("groq", cfg)
		}
		return nil
	}

	logger.LogInfo(fmt.Sprintf("[FACTORY_TEXT] Text client OK: %s/%s", llm.GetProvider(), llm.GetModel()))
	return llm
}

func NewLLMAudio(cfg *config.Config) AudioLLM {
	provider := strings.ToLower(strings.TrimSpace(cfg.LlmAudio))
	if provider == "" {
		provider = "groq" // groq whisper é o padrão de audio
	}

	logger.LogInfo(fmt.Sprintf("[FACTORY_AUDIO] Provider configurado: %s", provider))

	llm, err := newLLMClient(provider, cfg)
	if err != nil {
		logger.Error(
			context.Background(), fmt.Sprintf("[FACTORY_AUDIO] Falha ao criar client audio [%s]", provider),
			zap.Error(err),
		)
		return nil
	}

	logger.LogInfo(fmt.Sprintf("[FACTORY_AUDIO] Audio client OK: %s/%s", llm.GetProvider(), llm.GetModel()))
	return llm
}

func NewLLMVision(cfg *config.Config) VisionLLM {
	provider := strings.ToLower(strings.TrimSpace(cfg.LlmVision))
	if provider == "" {
		provider = "gemini" // gemini vision é o padrão
	}

	logger.LogInfo(fmt.Sprintf("[FACTORY_VISION] Provider configurado: %s", provider))

	llm, err := newLLMClient(provider, cfg)
	if err != nil {
		logger.Error(
			context.Background(), fmt.Sprintf("[FACTORY_VISION] Falha ao criar client vision [%s]", provider),
			zap.Error(err),
		)
		return nil
	}

	logger.LogInfo(fmt.Sprintf("[FACTORY_VISION] Vision client OK: %s/%s", llm.GetProvider(), llm.GetModel()))
	return llm
}

// NewAllClients - cria todos os clients disponíveis (usado pelo provider.go gestor)
func NewAllClients(cfg *config.Config) map[string]LLMClient {
	logger.LogInfo("[FACTORY] Criando pool de todos os clients disponíveis")

	clients := make(map[string]LLMClient)
	providers := []string{"groq", "gemini", "openai", "deepseek"}

	for _, p := range providers {
		client, err := newLLMClient(p, cfg)
		if err != nil {
			logger.Info(
				context.Background(), "Provider não disponível no pool",
				zap.String("provider", p),
				zap.Error(err),
			)
			continue
		}
		clients[p] = client
		logger.LogInfo(fmt.Sprintf("[FACTORY_POOL] Client %s adicionado ao pool", p))
	}

	logger.Info(
		context.Background(), "Pool de LLMs criado",
		zap.Int("total", len(clients)),
		zap.Any("providers", getKeys(clients)),
	)

	return clients
}

func getKeys(m map[string]LLMClient) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
