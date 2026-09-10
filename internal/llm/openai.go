package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/etoolstec/gokit/apperror"
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
		return "", apperror.NewInternalError("OPENAI_API_KEY não configurada", nil)
	}
	bodyReq := map[string]interface{}{
		"model": l.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
	}
	jb, _ := json.Marshal(bodyReq)
	req, err := http.NewRequestWithContext(ctx, "POST", l.baseURL, bytes.NewBuffer(jb))
	if err != nil {
		return "", mapLLMNetErr("openai", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", mapLLMNetErr("openai", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", mapLLMHTTPStatus("openai", resp.StatusCode, string(b))
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return "", apperror.NewInternalError("openai parse response", err)
	}
	if len(r.Choices) == 0 {
		return "", apperror.NewInternalError("sem resposta openai", nil)
	}
	return r.Choices[0].Message.Content, nil
}

func (l *OpenAILLM) TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error) {
	return "", apperror.NewBadRequestError("openai audio deve usar whisper dedicado - configure groq")
}

func (l *OpenAILLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	return "", apperror.NewBadRequestError("openai vision não implementado - use gemini")
}
