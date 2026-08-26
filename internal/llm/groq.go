package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

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

// DUMB CLIENT - só retorna string crua, provider faz parse
func (l *GroqLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	if l.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY não configurada")
	}
	url := fmt.Sprintf("%s/chat/completions", l.baseURL)
	bodyReq := map[string]interface{}{
		"model": l.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
	}
	jb, _ := json.Marshal(bodyReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("groq %d %s", resp.StatusCode, string(b))
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("sem resposta groq")
	}
	return r.Choices[0].Message.Content, nil
}

func (l *GroqLLM) TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error) {
	if l.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY não configurada")
	}
	if len(audio) == 0 {
		return "", fmt.Errorf("audio vazio")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "audio.ogg")
	part.Write(audio)
	_ = writer.WriteField("model", l.WhisperModel)
	_ = writer.WriteField("language", "pt")
	_ = writer.WriteField("response_format", "json")
	if prompt != "" {
		_ = writer.WriteField("prompt", prompt)
	}
	writer.Close()

	req, _ := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/audio/transcriptions", l.baseURL), body)
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("groq whisper %d: %s", resp.StatusCode, string(bb))
	}
	var r struct {
		Text string `json:"text"`
	}
	json.Unmarshal(bb, &r)
	return r.Text, nil
}

func (l *GroqLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	return "", fmt.Errorf("groq não suporta imagem - use gemini vision")
}
