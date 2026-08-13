// internal/llm/gemini.go
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

// GeminiLLM implementa LLMClient para Google Gemini
type GeminiLLM struct {
	apiKey  string
	model   string
	baseURL string
}

// NewGeminiLLM cria um novo cliente Gemini
func NewGeminiLLM(config *config.Config) LLMClient {
	model := config.LlmModel
	if model == "" {
		model = os.Getenv("GEMINI_MODEL")
	}
	if model == "" {
		model = "gemini-2.0-flash"
	}

	apiKey := config.LlmAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	return &GeminiLLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models",
	}
}

// Generate envia um prompt para a Gemini
func (llm *GeminiLLM) Generate(prompt string) (string, error) {
	return llm.GenerateWithContext(context.Background(), prompt)
}

// GenerateWithContext envia um prompt com contexto
func (llm *GeminiLLM) GenerateWithContext(ctx context.Context, prompt string) (string, error) {
	if llm.apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY não configurada")
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", llm.baseURL, llm.model, llm.apiKey)

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":      0.1,
			"responseMimeType": "application/json",
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error(ctx, "Erro ao serializar request body", zap.Error(err))
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.Error(ctx, "Erro ao criar request", zap.Error(err))
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(ctx, "Erro ao chamar Gemini API", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(ctx, "Erro ao ler resposta da Gemini", zap.Error(err))
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		// ✅ Log específico para erro 503
		if resp.StatusCode == 503 {
			logger.Warn(ctx, "Gemini API indisponível (503)",
				zap.String("model", llm.model),
				zap.String("response", string(body)),
			)
		} else {
			logger.Error(ctx, "Erro na Gemini API",
				zap.Int("status_code", resp.StatusCode),
				zap.String("response", string(body)),
			)
		}
		return "", fmt.Errorf("erro na API Gemini: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error(ctx, "Erro ao parsear resposta Gemini", zap.Error(err))
		return "", err
	}

	if len(result.Candidates) == 0 {
		logger.Warn(ctx, "Gemini retornou sem candidatos", zap.String("response", string(body)))
		return "", fmt.Errorf("nenhuma resposta da Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func (llm *GeminiLLM) GetModel() string {
	return llm.model
}

func (llm *GeminiLLM) GetProvider() string {
	return "gemini"
}

// ExtractIntent extrai a intenção do cliente da mensagem usando Gemini
func (llm *GeminiLLM) ExtractIntent(ctx context.Context, mensagem string, cardapio []dto.ProdutoItem) (*IntencaoCliente, error) {
	if llm.apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY não configurada")
	}

	// ✅ Log do início da extração
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

	// ✅ Configura e executa o retry com backoff
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

	// ✅ Função que será executada com retry
	var resposta string
	var err error

	resposta, err = RetryWithBackoff(ctx, config, func() (string, error) {
		return llm.GenerateWithContext(ctx, prompt)
	})

	if err != nil {
		// ✅ Log do erro
		logger.Error(ctx, "Falha na chamada Gemini após retries",
			zap.Error(err),
			zap.String("model", llm.GetModel()),
			zap.Int("max_attempts", config.MaxAttempts),
		)

		// ✅ Gera alerta (futura mensageria)
		gerarAlerta(ctx, "gemini", err)

		// ✅ Retorna erro amigável
		return nil, fmt.Errorf("serviço de IA temporariamente indisponível. Por favor, tente novamente em alguns segundos")
	}

	// ✅ Log de sucesso
	logger.Debug(ctx, "Resposta Gemini recebida",
		zap.String("provider", llm.GetProvider()),
		zap.String("model", llm.GetModel()),
		zap.Int("response_size", len(resposta)),
	)

	// Limpa a resposta
	resposta = cleanJSONResponse(resposta)

	// Parse do JSON
	var intencao IntencaoCliente
	if err := json.Unmarshal([]byte(resposta), &intencao); err != nil {
		if jsonStr := extractJSONFromText(resposta); jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &intencao); err != nil {
				logger.Error(ctx, "Erro ao parsear resposta JSON",
					zap.Error(err),
					zap.String("resposta", resposta),
				)
				return nil, fmt.Errorf("erro ao processar resposta da IA: %w", err)
			}
		} else {
			logger.Error(ctx, "Erro ao parsear resposta JSON", zap.Error(err))
			return nil, fmt.Errorf("erro ao processar resposta da IA: %w", err)
		}
	}

	// Valida a ação
	intencao.Acao = normalizeAcao(intencao.Acao)
	if intencao.Acao == "" {
		intencao.Acao = "visualizar"
	}

	// Se for adicionar ou remover, valida os itens contra o cardápio
	if intencao.Acao == "adicionar" || intencao.Acao == "remover" {
		for i, item := range intencao.Itens {
			encontrou, preco := itemExisteNoCardapio(cardapio, item.Nome)
			if !encontrou {
				similar := encontrarItemSimilar(cardapio, item.Nome)
				if similar != "" {
					intencao.Itens[i].Nome = similar
					logger.Info(ctx, "Item corrigido",
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

	logger.Info(ctx, "Intenção extraída com sucesso",
		zap.String("acao", intencao.Acao),
		zap.Int("itens_count", len(intencao.Itens)),
	)

	return &intencao, nil
}

// CorrigirNomes corrige nomes de produtos não encontrados
func (llm *GeminiLLM) CorrigirNomes(ctx context.Context, nomesNaoEncontrados []string, produtosEncontrados map[string]dto.ProdutoItem) ([]dto.ItemPedidoInput, error) {
	return CorrigirNomes(ctx, nomesNaoEncontrados, produtosEncontrados, llm.GenerateWithContext)
}

// ============================================
// ✅ FUNÇÃO DE ALERTA (Futura Mensageria)
// ============================================

// gerarAlerta gera um alerta para ser consumido por um sistema de mensageria
func gerarAlerta(ctx context.Context, provider string, err error) {
	// TODO: Implementar envio para sistema de mensageria (RabbitMQ, Kafka, SQS, etc)
	// Por enquanto, apenas loga o alerta

	logger.Warn(ctx, "🚨 ALERTA: Falha no provedor LLM",
		zap.String("provider", provider),
		zap.Error(err),
		zap.String("message", "Provedor LLM indisponível, verificar necessidade de fallback"),
	)

	// Estrutura do alerta para futura mensageria
	alerta := map[string]interface{}{
		"tipo":       "llm_unavailable",
		"provider":   provider,
		"erro":       err.Error(),
		"timestamp":  time.Now(),
		"severidade": "high",
		"acao":       "verificar disponibilidade do provedor e considerar fallback",
	}

	// TODO: Publicar alerta em fila
	// ex: messageQueue.Publish("alerts", alerta)

	// Para desenvolvimento, loga como JSON
	if alertaJSON, _ := json.Marshal(alerta); err == nil {
		logger.Debug(ctx, "Alerta JSON", zap.ByteString("alerta", alertaJSON))
	}
}
