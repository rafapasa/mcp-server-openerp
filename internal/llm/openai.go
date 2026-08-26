package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
)

type OpenAILLM struct {
	apiKey, model, baseURL string
}

func NewOpenAILLM(cfg *config.Config) LLMClient {
	model := cfg.LlmOpenAiModel
	if model == "" {
		model = "gpt-4o-mini"
	}
	if cfg.LlmOpenAiApiKey == "" {
		logger.Warn(context.Background(), "OPENAI_API_KEY não informada")
	}
	return &OpenAILLM{
		apiKey:  cfg.LlmOpenAiApiKey,
		model:   model,
		baseURL: "https://api.openai.com/v1/chat/completions",
	}
}

func (l *OpenAILLM) GetProvider() string { return "openai" }
func (l *OpenAILLM) GetModel() string    { return l.model }

func (l *OpenAILLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	if l.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY não configurada")
	}
	bodyReq := map[string]interface{}{
		"model": l.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
	}
	jb, _ := json.Marshal(bodyReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", l.baseURL, bytes.NewBuffer(jb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("openai %d: %s", resp.StatusCode, string(b))
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(b, &r)
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("sem resposta openai")
	}
	return r.Choices[0].Message.Content, nil
}

func (l *OpenAILLM) TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error) {
	return "", fmt.Errorf("openai audio deve usar whisper dedicado - configure groq")
}

func (l *OpenAILLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	// Se quiser usar vision da openai, implementa aqui, por enquanto delega erro
	return "", fmt.Errorf("openai vision não implementado - use gemini")
}
