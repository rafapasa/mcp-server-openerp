package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type DeepSeekLLM struct {
	apiKey, model, baseURL string
	cfg                    config.Config
}

func NewDeepSeekLLM(cfg *config.Config) LLMClient {
	model := cfg.LlmDeepSeekModel
	if model == "" {
		model = "deepseek-chat"
	}
	apiKey := cfg.LlmDeepSeekApiKey
	if apiKey == "" {
		logger.Warn(context.Background(), "API_KEY para DeepSeek não informada")
	}
	return &DeepSeekLLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.deepseek.com",
		cfg:     *cfg,
	}
}

func (llm *DeepSeekLLM) Generate(p string) (string, error) {
	return llm.GenerateWithContext(context.Background(), p)
}
func (llm *DeepSeekLLM) GetModel() string    { return llm.model }
func (llm *DeepSeekLLM) GetProvider() string { return "deepseek" }

func (llm *DeepSeekLLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY não configurada")
	}
	url := fmt.Sprintf("%s/chat/completions", llm.baseURL)
	bodyReq := map[string]interface{}{
		"model":       llm.model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.1, "response_format": map[string]string{"type": "json_object"}, "stream": false,
	}
	jb, _ := json.Marshal(bodyReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", llm.apiKey))
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
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

func (llm *DeepSeekLLM) ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*dto.IntencaoCliente, error) {
	catalogoStr := formatarCardapioParaPrompt(cardapio)
	prompt := fmt.Sprintf(PromptExtractIntentUniversal,
		"Estabelecimento", "geral", "misto",
		catalogoStr, mensagem, "",
	)
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
	// correção de nomes
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
	logger.Info(ctx, "Intenção deepseek", zap.String("acao", intencao.Acao), zap.Int("itens", len(intencao.Itens)))
	return &intencao, nil
}

func (llm *DeepSeekLLM) CorrigirNomes(ctx context.Context, n []string, p map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error) {
	return CorrigirNomes(ctx, n, p, llm.GenerateWithContext)
}

func (llm *DeepSeekLLM) GenerateWithAudio(ctx context.Context, promp string, audioBytes []byte) (string, error) {
	return "", fmt.Errorf(PromptAudioNotSupported, llm.GetProvider())
}

func (llm *DeepSeekLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	return llm.GenerateWithContext(ctx, prompt)
}

func (d *DeepSeekLLM) TranscribeAudio(ctx context.Context, audio []byte) (string, error) {
	// DeepSeek não transcreve, mas implementa pra não quebrar interface
	// Delega pro Groq se tiver key, senão erro controlado
	if groqKey := os.Getenv("GROQ_API_KEY"); groqKey != "" {
		g := NewGroqLLM(&d.cfg)
		return g.TranscribeAudio(ctx, audio)
	}
	return "", fmt.Errorf("deepseek não transcreve audio - configure GROQ_API_KEY")
}

func (d *DeepSeekLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	// mesma coisa - delega pro Gemini
	if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
		g := NewGeminiLLM(&d.cfg)
		return g.DescribeImage(ctx, image, prompt)
	}
	return "", fmt.Errorf("deepseek não descreve imagem - configure GEMINI_API_KEY")
}

func (d *DeepSeekLLM) GenerateWithImage(ctx context.Context, prompt, b64, mime string) (string, error) {
	return d.DescribeImage(ctx, []byte(b64), prompt)
}
