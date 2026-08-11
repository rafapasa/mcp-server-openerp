// Package llm provides language model clients and integrations.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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
}

// NewOpenAILLM cria um novo cliente OpenAI
func NewOpenAILLM(config *config.Config) LLMClient {
	baseURL := config.LlmBaseURL
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1/chat/completions"
	}

	model := config.LlmModel
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	apiKey := config.LlmAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	return &OpenAILLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

// Generate envia um prompt para a OpenAI
func (llm *OpenAILLM) Generate(prompt string) (string, error) {
	return llm.GenerateWithContext(context.Background(), prompt)
}

// GenerateWithContext envia um prompt com contexto
func (llm *OpenAILLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY não configurada")
	}

	requestBody := map[string]interface{}{
		"model": llm.model,
		"messages": []map[string]string{
			{"role": "system", "content": "Você é um assistente especializado em extrair pedidos de fast-food. Retorne apenas JSON válido."},
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

// ExtractIntent extrai a intenção do cliente da mensagem usando OpenAI
func (llm *OpenAILLM) ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*IntencaoCliente, error) {
	// A implementação é IDÊNTICA à do Gemini, apenas muda o cliente
	// Poderíamos extrair para uma função comum, mas vamos manter por enquanto
	logger.Debug(ctx, "Chamando LLM para extrair intenção", zap.String("provider", llm.GetProvider()), zap.String("model", llm.GetModel()), zap.Int("prompt_size", len(mensagem)))
	if llm.apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY não configurada")
	}

	cardapioStr := formatarCardapioParaPrompt(cardapio)

	prompt := fmt.Sprintf(`
Você é um assistente de atendimento especializado em interpretar pedidos de clientes para estabelecimentos comerciais (restaurantes, mercados, farmácias).

CARDÁPIO DISPONÍVEL:
%s

MENSAGEM DO CLIENTE:
"%s"

INSTRUÇÕES:
1. Identifique a INTENÇÃO do cliente com base na mensagem:
   - "adicionar" ou "add": cliente quer adicionar itens ao carrinho
   - "remover" ou "remove": cliente quer remover itens do carrinho
   - "finalizar" ou "confirmar" ou "fechar": cliente quer finalizar o pedido
   - "limpar" ou "clear": cliente quer limpar todo o carrinho
   - "visualizar": cliente quer ver o carrinho atual (padrão)

2. Se a intenção for "adicionar" ou "remover", extraia os itens mencionados:
   - nome: nome do item (use o nome exato do cardápio quando possível)
   - quantidade: número de unidades (padrão: 1)
   - observacao: observações como "sem cebola", "bem passado", etc.

3. Para "finalizar", "limpar" ou "visualizar", a lista de itens deve estar vazia.

4. Retorne APENAS o JSON, sem texto adicional.

FORMATO DE RESPOSTA (JSON):
{
  "acao": "adicionar",
  "itens": [
    {"nome": "X-Bacon", "quantidade": 2, "observacao": "sem cebola"}
  ],
  "mensagem": "quero um x-bacon e uma coca"
}
`, cardapioStr, mensagem)

	resposta, err := llm.GenerateWithContext(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar OpenAI: %w", err)
	}

	resposta = cleanJSONResponse(resposta)
	logger.Debug(ctx, "Resposta LLM recebida",
		zap.String("provider", llm.GetProvider()),
		zap.String("model", llm.GetModel()),
		zap.Int("response_size", len(resposta)))

	var intencao IntencaoCliente
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
					intencao.Itens[i].Nome = similar // Corrigido para usar o nome similar
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
