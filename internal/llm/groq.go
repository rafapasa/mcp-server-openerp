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
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type GroqLLM struct {
	apiKey       string
	model        string
	baseURL      string
	WhisperModel string
	httpClient   *http.Client
	cfg          *config.Config
}

func NewGroqLLM(cfg *config.Config) LLMClient { // mudou para AudioLLM para Wire diferenciar
	baseURL := "https://api.groq.com/openai/v1"
	model := cfg.LlmGroqModel
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}
	apiKey := cfg.LlmGroqApiKey
	if apiKey == "" {
		logger.LogInfo("GROQ_API_KEY não definida")
	}
	return &GroqLLM{
		apiKey:       apiKey,
		model:        model,
		baseURL:      baseURL,
		WhisperModel: cfg.LlmGroqWhisperModel,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
		cfg:          cfg,
	}
}

func (llm *GroqLLM) Generate(p string) (string, error) {
	return llm.GenerateWithContext(context.Background(), p)
}
func (llm *GroqLLM) GetModel() string    { return llm.model }
func (llm *GroqLLM) GetProvider() string { return "groq" }

func (llm *GroqLLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY não configurada")
	}
	url := fmt.Sprintf("%s/chat/completions", llm.baseURL)
	bodyReq := map[string]interface{}{
		"model": llm.model,
		"messages": []map[string]string{
			{"role": "system", "content": fmt.Sprintf(PromptSystemBaseShort, "Estabelecimento", "geral")},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1, "response_format": map[string]string{"type": "json_object"},
	}
	jb, _ := json.Marshal(bodyReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+llm.apiKey)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
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
	json.Unmarshal(b, &r)
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("sem resposta groq")
	}
	return r.Choices[0].Message.Content, nil
}

func (llm *GroqLLM) ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*dto.IntencaoCliente, error) {
	catalogoStr := formatarCardapioParaPrompt(cardapio)
	prompt := fmt.Sprintf(PromptExtractIntentUniversal, "Estab", "geral", "misto", catalogoStr, mensagem, "")
	resposta, err := RetryWithBackoff(ctx, RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Second, MaxDelay: 5 * time.Second, Backoff: func(a int) time.Duration { return time.Duration(1<<uint(a)) * time.Second }}, func() (string, error) { return llm.GenerateWithContext(ctx, prompt) })
	if err != nil {
		return nil, err
	}
	resposta = cleanJSONResponse(resposta)
	var intencao dto.IntencaoCliente
	if err := json.Unmarshal([]byte(resposta), &intencao); err != nil {
		if s := extractJSONFromText(resposta); s != "" {
			json.Unmarshal([]byte(s), &intencao)
		} else {
			return nil, err
		}
	}
	intencao.Acao = normalizeAcao(intencao.Acao)
	if intencao.Acao == "" {
		intencao.Acao = "visualizar"
	}
	for i, it := range intencao.Itens {
		if ok, preco := itemExisteNoCardapio(cardapio, it.Nome); !ok {
			if sim := encontrarItemSimilar(cardapio, it.Nome); sim != "" {
				intencao.Itens[i].Nome = sim
			}
		} else {
			intencao.Itens[i].PrecoUnitario = preco
		}
		if intencao.Itens[i].Quantidade <= 0 {
			intencao.Itens[i].Quantidade = 1
		}
	}
	if intencao.Acao == "adicionar" {
		intencao.Itens = mergeItens(intencao.Itens)
	}
	logger.Info(ctx, "Intenção Groq", zap.String("acao", intencao.Acao))
	return &intencao, nil
}

func (llm *GroqLLM) CorrigirNomes(ctx context.Context, n []string, p map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error) {
	return CorrigirNomes(ctx, n, p, llm.GenerateWithContext)
}
func (llm *GroqLLM) GenerateWithImage(ctx context.Context, prompt, b64Data, mimeType string) (string, error) {
	return "", fmt.Errorf(PromptVisionNotSupported, llm.GetProvider())
}

// ÚNICA FUNÇÃO DE ÁUDIO - unificada
func (llm *GroqLLM) GenerateWithAudio(ctx context.Context, promp string, audioBytes []byte) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY não configurada")
	}
	if len(audioBytes) == 0 {
		return "", fmt.Errorf("audio vazio")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "audio.ogg")
	part.Write(audioBytes)
	_ = writer.WriteField("model", llm.WhisperModel)
	_ = writer.WriteField("language", "pt")
	_ = writer.WriteField("response_format", "json")
	if promp != "" {
		_ = writer.WriteField("prompt", promp)
	}
	writer.Close()

	req, _ := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/audio/transcriptions", llm.baseURL), body)
	req.Header.Set("Authorization", "Bearer "+llm.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := llm.httpClient.Do(req)
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

func (llm *GroqLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	return "", fmt.Errorf("Descrição de imagens não disponivel para Groq")
}

func (llm *GroqLLM) TranscribeAudio(ctx context.Context, audio []byte) (string, error) {
	return llm.GenerateWithAudio(ctx, PromptGenerateWithAudio, audio)
}

func (llm *GroqLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	return llm.GenerateWithContext(ctx, PromptGenerateWithAudio)
}
