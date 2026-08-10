// internal/llm/gemini.go
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

// GeminiLLM implementa LLMClient para Google Gemini
type GeminiLLM struct {
	apiKey  string
	model   string
	baseURL string
}

// NewGeminiLLM cria um novo cliente Gemini
func NewGeminiLLM(config Config) *GeminiLLM {
	model := config.Model
	if model == "" {
		model = os.Getenv("GEMINI_MODEL")
	}
	if model == "" {
		model = "gemini-3.6-flash"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	return &GeminiLLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models",
	}
}

// Generate envia um prompt para a Gemini
func (llm *GeminiLLM) Generate(prompt string) (string, error) {
	return llm.GenerateWithContext(context.Background(), prompt)
}

// GenerateWithContext envia um prompt com contexto
func (llm *GeminiLLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY não configurada")
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", llm.baseURL, llm.model, llm.apiKey)

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":      0.1,
			"responseMimeType": "application/json",
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

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
		return "", fmt.Errorf("erro na API Gemini: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Candidates) == 0 {
		return "", fmt.Errorf("nenhuma resposta da Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func (llm *GeminiLLM) GetModel() string {
	return llm.model
}

func (llm *GeminiLLM) GetProvider() string {
	return "gemini"
}
