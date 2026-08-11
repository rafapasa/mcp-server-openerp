// Package llm provides LLM client implementations and shared configuration.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

// GroqLLM implementa LLMClient para Groq
type GroqLLM struct {
	apiKey  string
	model   string
	baseURL string
}

// NewGroqLLM cria um novo cliente Groq
func NewGroqLLM(config *config.Config) LLMClient {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("GROQ_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1/chat/completions"
	}

	model := config.Model
	if model == "" {
		model = os.Getenv("GROQ_MODEL")
	}
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GROQ_API_KEY")
	}

	return &GroqLLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

// Generate envia um prompt para a Groq
func (llm *GroqLLM) Generate(prompt string) (string, error) {
	return llm.GenerateWithContext(context.Background(), prompt)
}

// GenerateWithContext envia um prompt com contexto
func (llm *GroqLLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY não configurada")
	}

	requestBody := map[string]interface{}{
		"model": llm.model,
		"messages": []map[string]string{
			{"role": "system", "content": "Você é um assistente especializado em extrair pedidos de fast-food. Retorne apenas JSON válido."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", llm.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+llm.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erro na API Groq: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("nenhuma resposta da Groq")
	}

	return result.Choices[0].Message.Content, nil
}

// GetModel retorna o modelo configurado do cliente Groq.
func (llm *GroqLLM) GetModel() string {
	return llm.model
}

// GetProvider retorna o provedor de LLM.
func (llm *GroqLLM) GetProvider() string {
	return "groq"
}

func (llm *GroqLLM) ExtractIntent(mensagem string, cardapio []dto.ProdutoItem) (*IntencaoCliente, error) {
	// prompt := fmt.Sprintf
	return nil, nil
}

func (llm *GroqLLM) CorrigirNomes(nomesNaoEncontrados []string, produtosEncontrados map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error) {
	return llm.CorrigirNomes(nomesNaoEncontrados, produtosEncontrados)
}
