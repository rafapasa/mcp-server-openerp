// internal/llm/deepseek.go
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

// DeepSeekLLM implementa LLMClient para DeepSeek
type DeepSeekLLM struct {
	apiKey  string
	model   string
	baseURL string
}

// NewDeepSeekLLM cria um novo cliente DeepSeek
func NewDeepSeekLLM(cfg *config.Config) LLMClient {
	model := cfg.LlmModel
	if model == "" {
		model = os.Getenv("DEEPSEEK_MODEL")
	}
	if model == "" {
		model = os.Getenv("LLM_MODEL")
	}
	if model == "" {
		model = "deepseek-chat" // v4-flash
	}

	apiKey := cfg.LlmAPIKey
	if apiKey == "" {
		apiKey = cfg.LlmAPIKey
	}
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}

	return &DeepSeekLLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.deepseek.com",
	}
}

func (llm *DeepSeekLLM) Generate(prompt string) (string, error) {
	return llm.GenerateWithContext(context.Background(), prompt)
}

func (llm *DeepSeekLLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY não configurada")
	}

	url := fmt.Sprintf("%s/chat/completions", llm.baseURL)

	requestBody := map[string]interface{}{
		"model": llm.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
		"response_format": map[string]string{
			"type": "json_object",
		},
		"stream": false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error(ctx, "Erro ao serializar request body DeepSeek", zap.Error(err))
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.Error(ctx, "Erro ao criar request DeepSeek", zap.Error(err))
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", llm.apiKey))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(ctx, "Erro ao chamar DeepSeek API", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(ctx, "Erro ao ler resposta da DeepSeek", zap.Error(err))
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 429 || resp.StatusCode == 503 {
			logger.Warn(ctx, "DeepSeek API indisponível",
				zap.String("model", llm.model),
				zap.Int("status", resp.StatusCode),
				zap.String("response", string(body)),
			)
		} else {
			logger.Error(ctx, "Erro na DeepSeek API",
				zap.Int("status_code", resp.StatusCode),
				zap.String("response", string(body)),
			)
		}
		return "", fmt.Errorf("erro na API DeepSeek: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error(ctx, "Erro ao parsear resposta DeepSeek", zap.Error(err))
		return "", err
	}

	if len(result.Choices) == 0 {
		logger.Warn(ctx, "DeepSeek retornou sem choices", zap.String("response", string(body)))
		return "", fmt.Errorf("nenhuma resposta da DeepSeek")
	}

	return result.Choices[0].Message.Content, nil
}

func (llm *DeepSeekLLM) GetModel() string {
	return llm.model
}

func (llm *DeepSeekLLM) GetProvider() string {
	return "deepseek"
}

func (llm *DeepSeekLLM) ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*IntencaoCliente, error) {
	if llm.apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY não configurada")
	}

	logger.Debug(ctx, "Iniciando extração de intenção",
		zap.String("provider", llm.GetProvider()),
		zap.String("model", llm.GetModel()),
		zap.Int("cardapio_size", len(cardapio)),
	)

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

EXEMPLOS:
- "quero um x-bacon e uma coca" → {"acao": "adicionar", "itens": [{"nome":"X-Bacon","quantidade":1}, {"nome":"Coca-Cola","quantidade":1}], "mensagem": "..."}
- "remove a coca" → {"acao": "remover", "itens": [{"nome":"Coca-Cola","quantidade":1}], "mensagem": "..."}
- "finalizar pedido" → {"acao": "finalizar", "itens": [], "mensagem": "..."}
- "limpar carrinho" → {"acao": "limpar", "itens": [], "mensagem": "..."}
- "mostrar carrinho" → {"acao": "visualizar", "itens": [], "mensagem": "..."}
`, cardapioStr, mensagem)

	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    10 * time.Second,
		Backoff: func(attempt int) time.Duration {
			delay := time.Duration(1<<uint(attempt)) * time.Second
			if delay > 10*time.Second {
				delay = 10 * time.Second
			}
			return delay
		},
	}

	var resposta string
	var err error

	resposta, err = RetryWithBackoff(ctx, config, func() (string, error) {
		return llm.GenerateWithContext(ctx, prompt)
	})

	if err != nil {
		logger.Error(ctx, "Falha na chamada DeepSeek após retries",
			zap.Error(err),
			zap.String("model", llm.GetModel()),
			zap.Int("max_attempts", config.MaxAttempts),
		)
		gerarAlerta(ctx, "deepseek", err)
		return nil, fmt.Errorf("serviço de IA temporariamente indisponível. Por favor, tente novamente em alguns segundos")
	}

	logger.Debug(ctx, "Resposta DeepSeek recebida",
		zap.String("provider", llm.GetProvider()),
		zap.String("model", llm.GetModel()),
		zap.Int("response_size", len(resposta)),
	)

	resposta = cleanJSONResponse(resposta)

	var intencao IntencaoCliente
	if err := json.Unmarshal([]byte(resposta), &intencao); err != nil {
		if jsonStr := extractJSONFromText(resposta); jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &intencao); err != nil {
				logger.Error(ctx, "Erro ao parsear resposta JSON DeepSeek",
					zap.Error(err),
					zap.String("resposta", resposta),
				)
				return nil, fmt.Errorf("erro ao processar resposta da IA: %w", err)
			}
		} else {
			logger.Error(ctx, "Erro ao parsear resposta JSON DeepSeek", zap.Error(err))
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
					logger.Info(ctx, "Item corrigido DeepSeek",
						zap.String("original", item.Nome),
						zap.String("corrigido", similar),
					)
				} else {
					logger.Warn(ctx, "Item não encontrado no cardápio",
						zap.String("item_nome", item.Nome),
					)
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

	logger.Info(ctx, "Intenção extraída com sucesso DeepSeek",
		zap.String("acao", intencao.Acao),
		zap.Int("itens_count", len(intencao.Itens)),
	)

	return &intencao, nil
}

func (llm *DeepSeekLLM) CorrigirNomes(ctx context.Context, nomesNaoEncontrados []string, produtosEncontrados map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error) {
	return CorrigirNomes(ctx, nomesNaoEncontrados, produtosEncontrados, llm.GenerateWithContext)
}

func (llm *DeepSeekLLM) GenerateWithImage(ctx context.Context, prompt, b64Data, mimeType string) (string, error) {
	// DeepSeek não tem vision nativo barato, delega pro texto
	return "", fmt.Errorf("vision não suportado no provider %s, use gemini", llm.GetProvider())
}
