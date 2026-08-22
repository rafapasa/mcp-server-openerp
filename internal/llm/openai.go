// Package llm provides language model clients and integrations.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

// OpenAILLM implementa LLMClient para OpenAI
type OpenAILLM struct {
	apiKey  string
	model   string
	baseURL string
	cfg     *config.Config
}

// NewOpenAILLM cria um novo cliente OpenAI
func NewOpenAILLM(cfg *config.Config) LLMClient {
	baseURL := "https://api.openai.com/v1/chat/completions"

	model := cfg.LlmOpenAiModel
	if model == "" {
		model = "gpt-4o-mini"
	}

	apiKey := cfg.LlmOpenAiApiKey
	if apiKey == "" {
		logger.Warn(context.Background(), "API_KEY para OpemIA não informada")
	}

	return &OpenAILLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		cfg:     cfg,
	}
}

func (llm *OpenAILLM) Generate(prompt string) (string, error) {
	return llm.GenerateWithContext(context.Background(), prompt)
}

func (llm *OpenAILLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY não configurada")
	}

	// Usa system prompt universal curto
	systemPrompt := fmt.Sprintf(PromptSystemBaseShort, "Estabelecimento", "geral")

	requestBody := map[string]interface{}{
		"model": llm.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.1,
		"response_format": map[string]string{"type": "json_object"},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", llm.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+llm.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erro na API OpenAI: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("nenhuma resposta da OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

func (llm *OpenAILLM) GetModel() string {
	return llm.model
}

func (llm *OpenAILLM) GetProvider() string {
	return "openai"
}

func (llm *OpenAILLM) ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*dto.IntencaoCliente, error) {
	if llm.apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY não configurada")
	}

	logger.Debug(ctx, "Chamando LLM para extrair intenção",
		zap.String("provider", llm.GetProvider()),
		zap.String("model", llm.GetModel()),
		zap.Int("prompt_size", len(mensagem)),
	)

	catalogoStr := formatarCardapioParaPrompt(cardapio)

	// Usa prompt universal - tenant genérico aqui, o service que chama deve passar tenant real no futuro
	// Para manter assinatura atual, usamos placeholders genéricos
	prompt := fmt.Sprintf(PromptExtractIntentUniversal,
		"Estabelecimento", // Nome
		"geral",           // Nicho - vai ser substituído por tenant.Segmento quando refatorar service
		"misto",           // Tipo entrega
		catalogoStr,
		mensagem,
		"", // histórico vazio - service pode injetar depois
	)

	resposta, err := llm.GenerateWithContext(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar OpenAI: %w", err)
	}

	resposta = cleanJSONResponse(resposta)
	logger.Debug(ctx, "Resposta LLM recebida",
		zap.String("provider", llm.GetProvider()),
		zap.String("model", llm.GetModel()),
		zap.Int("response_size", len(resposta)))

	var intencao dto.IntencaoCliente
	if err := json.Unmarshal([]byte(resposta), &intencao); err != nil {
		if jsonStr := extractJSONFromText(resposta); jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &intencao); err != nil {
				return nil, fmt.Errorf("erro ao parsear resposta da IA: %v\nResposta: %s", err, resposta)
			}
		} else {
			return nil, fmt.Errorf("erro ao parsear resposta da IA: %v\nResposta: %s", err, resposta)
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
					logger.Info(ctx, "Item corrigido", zap.String("original", item.Nome), zap.String("corrigido", similar))
				}
				logger.Warn(ctx, "Item não encontrado no cardápio", zap.String("item_nome", item.Nome))
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

	logger.Info(ctx, "Intenção extraída", zap.String("acao", intencao.Acao), zap.Int("itens", len(intencao.Itens)))

	return &intencao, nil
}

func (llm *OpenAILLM) CorrigirNomes(ctx context.Context, nomesNaoEncontrados []string, produtosEncontrados map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error) {
	return CorrigirNomes(ctx, nomesNaoEncontrados, produtosEncontrados, llm.GenerateWithContext)
}

func (llm *OpenAILLM) GenerateWithImage(ctx context.Context, prompt, b64Data, mimeType string) (string, error) {
	return "", fmt.Errorf(PromptVisionNotSupported, llm.GetProvider())
}

func (llm *OpenAILLM) GenerateWithAudio(ctx context.Context, promp string, audioBytes []byte) (string, error) {
	return "", fmt.Errorf(PromptAudioNotSupported, llm.GetProvider())
}

func (llm *OpenAILLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	return llm.GenerateWithContext(ctx, PromptGenerateWithAudio)
}

func (llm *OpenAILLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	return llm.GenerateWithImage(ctx, prompt, string(image), "")
}

func (llm *OpenAILLM) TranscribeAudio(ctx context.Context, audio []byte) (string, error) {
	return llm.GenerateWithAudio(ctx, PromptGenerateWithAudio, audio)
}
