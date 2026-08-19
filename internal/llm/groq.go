// internal/llm/groq.go
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

type GroqLLM struct {
	apiKey  string
	model   string
	baseURL string
}

func NewGroqLLM(cfg *config.Config) LLMClient {
	baseURL := cfg.LlmBaseURL
	if baseURL == "" {
		baseURL = os.Getenv("GROQ_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}

	model := cfg.LlmModel
	if model == "" {
		model = os.Getenv("GROQ_MODEL")
	}
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	apiKey := cfg.LlmAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GROQ_API_KEY")
	}

	return &GroqLLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

func (llm *GroqLLM) Generate(prompt string) (string, error) {
	return llm.GenerateWithContext(context.Background(), prompt)
}

func (llm *GroqLLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY não configurada")
	}

	url := fmt.Sprintf("%s/chat/completions", llm.baseURL)

	requestBody := map[string]interface{}{
		"model": llm.model,
		"messages": []map[string]string{
			{"role": "system", "content": "Você é um assistente especializado em extrair pedidos. Retorne apenas JSON válido."},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.1,
		"response_format": map[string]string{"type": "json_object"},
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+llm.apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(ctx, "Erro ao chamar Groq API", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		logger.Error(ctx, "Erro na Groq API", zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return "", fmt.Errorf("erro na API Groq: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(body, &result)
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("nenhuma resposta da Groq")
	}
	return result.Choices[0].Message.Content, nil
}

func (llm *GroqLLM) GetModel() string    { return llm.model }
func (llm *GroqLLM) GetProvider() string { return "groq" }

func (llm *GroqLLM) ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*IntencaoCliente, error) {
	if llm.apiKey == "" {
		return nil, fmt.Errorf("GROQ_API_KEY não configurada")
	}

	logger.Debug(ctx, "Iniciando extração Groq", zap.String("model", llm.model), zap.Int("cardapio", len(cardapio)))

	cardapioStr := formatarCardapioParaPrompt(cardapio)

	prompt := fmt.Sprintf(`
Você é um assistente de atendimento especializado em interpretar pedidos de clientes para estabelecimentos comerciais (restaurantes, mercados, farmácias).

CARDÁPIO DISPONÍVEL:
%s

MENSAGEM DO CLIENTE:
"%s"

INSTRUÇÕES:
1. Identifique a INTENÇÃO: "adicionar", "remover", "finalizar", "limpar", "visualizar"
2. Se for adicionar/remover, extraia itens: nome (exato do cardápio quando possível), quantidade, observacao
3. Retorne APENAS JSON.

FORMATO:
{
  "acao": "adicionar",
  "itens": [{"nome": "X-Bacon", "quantidade": 2, "observacao": "sem cebola"}],
  "mensagem": "quero um x-bacon e uma coca"
}
`, cardapioStr, mensagem)

	retryCfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    5 * time.Second,
		Backoff:     func(a int) time.Duration { return time.Duration(1<<uint(a)) * time.Second },
	}

	resposta, err := RetryWithBackoff(ctx, retryCfg, func() (string, error) {
		return llm.GenerateWithContext(ctx, prompt)
	})
	if err != nil {
		logger.Error(ctx, "Falha Groq após retries", zap.Error(err))
		gerarAlerta(ctx, "groq", err)
		return nil, fmt.Errorf("serviço de IA temporariamente indisponível")
	}

	resposta = cleanJSONResponse(resposta)
	var intencao IntencaoCliente
	if err := json.Unmarshal([]byte(resposta), &intencao); err != nil {
		if s := extractJSONFromText(resposta); s != "" {
			json.Unmarshal([]byte(s), &intencao)
		} else {
			return nil, fmt.Errorf("erro ao processar resposta da IA: %w", err)
		}
	}

	intencao.Acao = normalizeAcao(intencao.Acao)
	if intencao.Acao == "" {
		intencao.Acao = "visualizar"
	}

	if intencao.Acao == "adicionar" || intencao.Acao == "remover" {
		for i, item := range intencao.Itens {
			encontrou, preco := itemExisteNoCardapio(cardapio, item.Nome)
			if !encontrou {
				similar := encontrarItemSimilar(cardapio, item.Nome)
				if similar != "" {
					intencao.Itens[i].Nome = similar
				}
			} else {
				intencao.Itens[i].PrecoUnitario = preco
			}
			if intencao.Itens[i].Quantidade <= 0 {
				intencao.Itens[i].Quantidade = 1
			}
		}
	}
	if intencao.Acao == "adicionar" {
		intencao.Itens = mergeItens(intencao.Itens)
	}

	logger.Info(ctx, "Intenção extraída Groq", zap.String("acao", intencao.Acao), zap.Int("itens", len(intencao.Itens)))
	return &intencao, nil
}

func (llm *GroqLLM) CorrigirNomes(ctx context.Context, naoEncontrados []string, encontrados map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error) {
	return CorrigirNomes(ctx, naoEncontrados, encontrados, llm.GenerateWithContext)
}

// TranscribeAudio - só Groq tem isso barato
func (llm *GroqLLM) TranscribeAudio(ctx context.Context, audioBytes []byte) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY não configurada")
	}

	url := fmt.Sprintf("%s/audio/transcriptions", llm.baseURL)

	body := &bytes.Buffer{}
	writer := io.MultiWriter(body)
	// multipart form
	boundary := "----groqboundary"
	// simplificado - usa mime/multipart na implementação real
	// aqui vai o arquivo: file, model=whisper-large-v3, language=pt

	req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
	req.Header.Set("Authorization", "Bearer "+llm.apiKey)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	_ = writer

	// TODO: implementação multipart real, ou use openai-go pra whisper
	return "", nil
}

func (llm *GroqLLM) GenerateWithImage(ctx context.Context, prompt, b64Data, mimeType string) (string, error) {
	return "", fmt.Errorf("vision não suportado no provider %s, use gemini", llm.GetProvider())
}
