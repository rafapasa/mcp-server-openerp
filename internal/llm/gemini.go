package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
)

type GeminiLLM struct {
	apiKey, model, baseURL string
}

func NewGeminiLLM(cfg *config.Config) LLMClient {
	model := cfg.LlmGeminiModel
	if model == "" {
		model = "gemini-2.0-flash"
	}
	if cfg.LlmGeminiApiKey == "" {
		logger.Warn(context.Background(), "GEMINI_API_KEY não informada")
	}
	return &GeminiLLM{
		apiKey:  cfg.LlmGeminiApiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models",
	}
}

func (l *GeminiLLM) GetProvider() string { return "gemini" }
func (l *GeminiLLM) GetModel() string    { return l.model }

func (l *GeminiLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	if l.apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY não configurada")
	}
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", l.baseURL, l.model, l.apiKey)
	bodyReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.1,
		},
	}
	jb, _ := json.Marshal(bodyReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gemini %d: %s", resp.StatusCode, string(b))
	}
	var r struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	json.Unmarshal(b, &r)
	if len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("sem resposta gemini")
	}
	return r.Candidates[0].Content.Parts[0].Text, nil
}

func (l *GeminiLLM) TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error) {
	if len(audio) == 0 {
		return "", fmt.Errorf("audio vazio")
	}
	if prompt == "" {
		prompt = PromptTranscribeSimple
	}
	b64 := base64.StdEncoding.EncodeToString(audio)
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", l.baseURL, l.model, l.apiKey)
	bodyReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]interface{}{
				{"text": prompt},
				{"inline_data": map[string]string{"mime_type": "audio/ogg", "data": b64}},
			}},
		},
		"generationConfig": map[string]interface{}{"temperature": 0.2},
	}
	jb, _ := json.Marshal(bodyReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gemini audio %d %s", resp.StatusCode, string(b))
	}
	var r struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	json.Unmarshal(b, &r)
	if len(r.Candidates) == 0 {
		return "", fmt.Errorf("sem resposta audio gemini")
	}
	return r.Candidates[0].Content.Parts[0].Text, nil
}

func (l *GeminiLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	if prompt == "" {
		prompt = PromptVisionDescribe
	}
	// image vem como bytes crus, converte pra base64
	b64 := base64.StdEncoding.EncodeToString(image)
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", l.baseURL, l.model, l.apiKey)
	bodyReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]interface{}{
				{"text": prompt},
				{"inline_data": map[string]string{"mime_type": "image/jpeg", "data": b64}},
			}},
		},
		"generationConfig": map[string]interface{}{"temperature": 0.2},
	}
	jb, _ := json.Marshal(bodyReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 45 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gemini vision %d %s", resp.StatusCode, string(b))
	}
	var r struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	json.Unmarshal(b, &r)
	if len(r.Candidates) == 0 {
		return "", fmt.Errorf("sem resposta vision")
	}
	return r.Candidates[0].Content.Parts[0].Text, nil
}
