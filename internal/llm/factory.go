// internal/llm/factory.go
package llm

import (
	"fmt"
	"log"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
)

// NewLLMClient cria um cliente LLM baseado na configuração
func NewLLMClient(config *config.Config) (LLMClient, error) {
	provider := strings.ToLower(config.LlmProvider)

	log.Printf("[LLM] Inicializando provedor: %s", provider)

	switch provider {
	case "openai":
		return NewOpenAILLM(config), nil
	case "groq":
		return NewGroqLLM(config), nil
	case "gemini":
		return NewGeminiLLM(config), nil
	default:
		return nil, fmt.Errorf("provedor LLM não suportado: %s", provider)
	}
}

// NewLLMClientFromEnv cria um cliente LLM usando variáveis de ambiente
func NewLLMClientFromEnv() (LLMClient, error) {
	config, _ := config.LoadConfig()
	return NewLLMClient(config)
}
