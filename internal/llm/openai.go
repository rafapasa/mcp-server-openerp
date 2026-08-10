// internal/llm/openai.go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// OpenAILLM implementa LLMClient para OpenAI
type OpenAILLM struct {
	apiKey  string
	model   string
	baseURL string
}

// NewOpenAILLM cria um novo cliente OpenAI
func NewOpenAILLM(config Config) *OpenAILLM {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1/chat/completions"
	}

	model := config.Model
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	return &OpenAILLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

// Generate envia um prompt para a OpenAI
func (llm *OpenAILLM) Generate(prompt string) (string, error) {
	return llm.GenerateWithContext(context.Background(), prompt)
}

// GenerateWithContext envia um prompt com contexto
func (llm *OpenAILLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY não configurada")
	}

	requestBody := map[string]interface{}{
		"model": llm.model,
		"messages": []map[string]string{
			{"role": "system", "content": "Você é um assistente especializado em extrair pedidos de fast-food. Retorne apenas JSON válido."},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.1,
		"response_format": map[string]string{"type": "json_object"},
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
		return "", fmt.Errorf("erro na API OpenAI: %d - %s", resp.StatusCode, string(body))
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
		return "", fmt.Errorf("nenhuma resposta da OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

func (llm *OpenAILLM) GetModel() string {
	return llm.model
}

func (llm *OpenAILLM) GetProvider() string {
	return "openai"
}
