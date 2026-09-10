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
		return "", apperror.NewInternalError("DEEPSEEK_API_KEY não configurada", nil)
	}
	url := l.baseURL + "/chat/completions"
	bodyReq := map[string]interface{}{
		"model":       l.model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.1,
		"stream":      false,
	}
	jb, _ := json.Marshal(bodyReq)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	if err != nil {
		return "", mapLLMNetErr("deepseek", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", mapLLMNetErr("deepseek", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", mapLLMHTTPStatus("deepseek", resp.StatusCode, string(b))
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return "", apperror.NewInternalError("deepseek parse response", err)
	}
	if len(r.Choices) == 0 {
		return "", apperror.NewInternalError("sem resposta deepseek", nil)
	}
	return r.Choices[0].Message.Content, nil
}

func (l *DeepSeekLLM) TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error) {
	return "", apperror.NewBadRequestError("deepseek não transcreve audio")
}

func (l *DeepSeekLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	return "", apperror.NewBadRequestError("deepseek não descreve imagem")
}
