package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/etoolstec/gokit/apperror"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
)

type GroqLLM struct {
	apiKey       string
	model        string
	baseURL      string
	WhisperModel string
	httpClient   *http.Client
}

func NewGroqLLM(cfg *config.Config) LLMClient {
	model := cfg.LlmGroqModel
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}
	whisper := cfg.LlmGroqWhisperModel
	if whisper == "" {
		whisper = "whisper-large-v3"
	}
	if cfg.LlmGroqApiKey == "" {
		logger.LogInfo("[GROQ] GROQ_API_KEY não definida")
	}
	return &GroqLLM{
		apiKey:       cfg.LlmGroqApiKey,
		model:        model,
		baseURL:      "https://api.groq.com/openai/v1",
		WhisperModel: whisper,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (l *GroqLLM) GetProvider() string { return "groq" }
func (l *GroqLLM) GetModel() string    { return l.model }

func (l *GroqLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	if l.apiKey == "" {
		return "", apperror.NewInternalError("GROQ_API_KEY não configurada", nil)
	}
	url := l.baseURL + "/chat/completions"
	bodyReq := map[string]interface{}{
		"model": l.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
	}
	jb, _ := json.Marshal(bodyReq)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	if err != nil {
		return "", mapLLMNetErr("groq", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", mapLLMNetErr("groq", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", mapLLMHTTPStatus("groq", resp.StatusCode, string(b))
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return "", apperror.NewInternalError("groq parse response", err)
	}
	if len(r.Choices) == 0 {
		return "", apperror.NewInternalError("sem resposta groq", nil)
	}
	return r.Choices[0].Message.Content, nil
}

func (l *GroqLLM) TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error) {
	if l.apiKey == "" {
		return "", apperror.NewInternalError("GROQ_API_KEY não configurada", nil)
	}
	if len(audio) == 0 {
		return "", apperror.NewBadRequestError("audio vazio")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "audio.ogg")
	if err != nil {
		return "", apperror.NewInternalError("groq form file", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", apperror.NewInternalError("groq write audio", err)
	}
	_ = writer.WriteField("model", l.WhisperModel)
	_ = writer.WriteField("language", "pt")
	_ = writer.WriteField("response_format", "json")
	if prompt != "" {
		_ = writer.WriteField("prompt", prompt)
	}
	_ = writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", l.baseURL+"/audio/transcriptions", body)
	if err != nil {
		return "", mapLLMNetErr("groq whisper", err)
	}
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", mapLLMNetErr("groq whisper", err)
	}
	defer resp.Body.Close()
	bb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", mapLLMHTTPStatus("groq whisper", resp.StatusCode, string(bb))
	}
	var r struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(bb, &r); err != nil {
		return "", apperror.NewInternalError("groq whisper parse", err)
	}
	return r.Text, nil
}

func (l *GroqLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	return "", apperror.NewBadRequestError("groq não suporta imagem - use gemini vision")
}
