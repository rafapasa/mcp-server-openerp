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
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
)

type GeminiLLM struct {
	apiKey, model, baseURL string
	cfg                    *config.Config
}

func NewGeminiLLM(cfg *config.Config) LLMClient {
	model := cfg.LlmGeminiModel
	if model == "" {
		model = "gemini-2.0-flash"
	}
	apiKey := cfg.LlmGeminiApiKey
	if apiKey == "" {
		logger.Warn(context.Background(), "API_KEY para Gemini não informada")
	}
	return &GeminiLLM{
		apiKey:  cfg.LlmGeminiApiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models",
		cfg:     cfg,
	}
}

func (llm *GeminiLLM) Generate(p string) (string, error) {
	return llm.GenerateWithContext(context.Background(), p)
}
func (llm *GeminiLLM) GetModel() string    { return llm.model }
func (llm *GeminiLLM) GetProvider() string { return "gemini" }

func (llm *GeminiLLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY não configurada")
	}
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", llm.baseURL, llm.model, llm.apiKey)
	bodyReq := map[string]interface{}{
		"contents":         []map[string]interface{}{{"parts": []map[string]string{{"text": prompt}}}},
		"generationConfig": map[string]interface{}{"temperature": 0.1, "responseMimeType": "application/json"},
	}
	jb, _ := json.Marshal(bodyReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
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
	if len(r.Candidates) == 0 {
		return "", fmt.Errorf("sem resposta")
	}
	return r.Candidates[0].Content.Parts[0].Text, nil
}

func (llm *GeminiLLM) GenerateWithImage(ctx context.Context, prompt, b64Data, mimeType string) (string, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	if prompt == "" {
		prompt = fmt.Sprintf(PromptGenerateWithImage, "Estab", "geral", "Extraia dados", "", "")
	}
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", llm.baseURL, llm.model, llm.apiKey)
	bodyReq := map[string]interface{}{
		"contents":         []map[string]interface{}{{"parts": []map[string]interface{}{{"text": prompt}, {"inline_data": map[string]string{"mime_type": mimeType, "data": b64Data}}}}},
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
		return "", fmt.Errorf("vision %d %s", resp.StatusCode, string(b))
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
	return r.Candidates[0].Content.Parts[0].Text, nil
}

func (llm *GeminiLLM) GenerateWithAudio(ctx context.Context, promp string, audioBytes []byte) (string, error) {
	if len(audioBytes) == 0 {
		return "", fmt.Errorf("audio vazio")
	}
	b64 := base64.StdEncoding.EncodeToString(audioBytes)
	if promp == "" {
		promp = fmt.Sprintf(PromptGenerateWithAudio, "Estab", "geral", "geral", "")
	}
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", llm.baseURL, llm.model, llm.apiKey)
	bodyReq := map[string]interface{}{
		"contents":         []map[string]interface{}{{"parts": []map[string]interface{}{{"text": promp}, {"inline_data": map[string]string{"mime_type": "audio/ogg", "data": b64}}}}},
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
		return "", fmt.Errorf("audio %d %s", resp.StatusCode, string(b))
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
		return "", fmt.Errorf("sem resposta audio")
	}
	return r.Candidates[0].Content.Parts[0].Text, nil
}

func (llm *GeminiLLM) ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*dto.IntencaoCliente, error) {
	cardapioStr := formatarCardapioParaPrompt(cardapio)
	prompt := fmt.Sprintf(PromptExtractIntentUniversal, "Estab", "geral", "misto", cardapioStr, mensagem, "")
	resposta, err := RetryWithBackoff(ctx, RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Second, MaxDelay: 10 * time.Second, Backoff: func(a int) time.Duration {
		d := time.Duration(1<<uint(a)) * time.Second
		if d > 10*time.Second {
			d = 10 * time.Second
		}
		return d
	}}, func() (string, error) { return llm.GenerateWithContext(ctx, prompt) })
	if err != nil {
		return nil, err
	}
	resposta = cleanJSONResponse(resposta)
	var intencao dto.IntencaoCliente
	if err := json.Unmarshal([]byte(resposta), &intencao); err != nil {
		if js := extractJSONFromText(resposta); js != "" {
			json.Unmarshal([]byte(js), &intencao)
		} else {
			return nil, err
		}
	}
	intencao.Acao = normalizeAcao(intencao.Acao)
	if intencao.Acao == "" {
		intencao.Acao = "visualizar"
	}
	return &intencao, nil
}

func (llm *GeminiLLM) CorrigirNomes(ctx context.Context, n []string, p map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error) {
	return CorrigirNomes(ctx, n, p, llm.GenerateWithContext)
}

func (llm *GeminiLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	return llm.GenerateWithImage(ctx, prompt, string(image), "")
}

func (llm *GeminiLLM) TranscribeAudio(ctx context.Context, audio []byte) (string, error) {
	return llm.GenerateWithAudio(ctx, PromptGenerateWithAudio, audio)
}

func (llm *GeminiLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	return llm.GenerateWithContext(ctx, PromptGenerateWithAudio)
}
