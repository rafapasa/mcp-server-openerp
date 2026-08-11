package llm

import (
	"context"
	"os"

	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// LLMClient é a interface que todos os provedores de LLM devem implementar
type LLMClient interface {
	// Generate envia um prompt e retorna a resposta
	Generate(prompt string) (string, error)

	// GenerateWithContext envia um prompt com contexto e retorna a resposta
	GenerateWithContext(ctx context.Context, prompt string) (string, error)

	// GetModel retorna o nome do modelo sendo usado
	GetModel() string

	// GetProvider retorna o nome do provedor (openai, groq, gemini, etc)
	GetProvider() string

	// ExtractIntent extrai a intenção do cliente da mensagem
	ExtractIntent(mensagem string, cardapio []service.ProdutoItem) (*IntencaoCliente, error)
}

// IntencaoCliente representa a intenção extraída da mensagem
type IntencaoCliente struct {
	Acao     string                    `json:"acao"` // adicionar, remover, finalizar, visualizar, limpar
	Itens    []service.ItemPedidoInput `json:"itens,omitempty"`
	Mensagem string                    `json:"mensagem,omitempty"`
}

// Config contém as configurações comuns para todos os LLMs
type Config struct {
	Provider    string
	APIKey      string
	Model       string
	BaseURL     string
	MaxTokens   int
	Temperature float64
}

// DefaultConfig retorna a configuração padrão baseada nas variáveis de ambiente
func DefaultConfig() Config {
	return Config{
		Provider:    getEnv("LLM_PROVIDER", "openai"),
		APIKey:      getEnv("LLM_API_KEY", ""),
		Model:       getEnv("LLM_MODEL", "gpt-4o-mini"),
		BaseURL:     getEnv("LLM_BASE_URL", ""),
		MaxTokens:   4096,
		Temperature: 0.1,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
