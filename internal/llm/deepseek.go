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

type DeepSeekLLM struct {
	apiKey, model, baseURL string
}

func NewDeepSeekLLM(cfg *config.Config) LLMClient {
	model := cfg.LlmDeepSeekModel
	if model == "" {
		model = "deepseek-chat"
	}
	if cfg.LlmDeepSeekApiKey == "" {
		logger.Warn(context.Background(), "DEEPSEEK_API_KEY não informada")
	}
	return &DeepSeekLLM{
		apiKey:  cfg.LlmDeepSeekApiKey,
		model:   model,
		baseURL: "https://api.deepseek.com",
	}
}

func (l *DeepSeekLLM) GetProvider() string { return "deepseek" }
func (l *DeepSeekLLM) GetModel() string    { return l.model }

func (l *DeepSeekLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	if l.apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY não configurada")
	}
	url := fmt.Sprintf("%s/chat/completions", l.baseURL)
	bodyReq := map[string]interface{}{
		"model":       l.model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.1,
		"stream":      false,
	}
	jb, _ := json.Marshal(bodyReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", l.apiKey))
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("deepseek %d: %s", resp.StatusCode, string(b))
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
		return "", fmt.Errorf("sem resposta deepseek")
	}
	return r.Choices[0].Message.Content, nil
}

func (l *DeepSeekLLM) TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error) {
	return "", fmt.Errorf("deepseek não transcreve audio")
}

func (l *DeepSeekLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	return "", fmt.Errorf("deepseek não descreve imagem")
}
