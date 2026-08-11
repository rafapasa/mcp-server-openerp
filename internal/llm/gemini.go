// internal/llm/gemini.go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// GeminiLLM implementa LLMClient para Google Gemini
type GeminiLLM struct {
	apiKey  string
	model   string
	baseURL string
}

// NewGeminiLLM cria um novo cliente Gemini
func NewGeminiLLM(config Config) LLMClient {
	model := config.Model
	if model == "" {
		model = os.Getenv("GEMINI_MODEL")
	}
	if model == "" {
		model = "gemini-3.6-flash"
	}

	apiKey := config.APIKey
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
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

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
		return "", err
	}

	if len(result.Candidates) == 0 {
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
func (llm *GeminiLLM) ExtractIntent(mensagem string, cardapio []service.ProdutoItem) (*IntencaoCliente, error) {
	if llm.apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY não configurada")
	}

	// Formata o cardápio para o prompt
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

	// Chama a API Gemini
	resposta, err := llm.GenerateWithContext(context.Background(), prompt)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar Gemini: %w", err)
	}

	// Limpa a resposta
	resposta = cleanJSONResponse(resposta)

	// Parse do JSON
	var intencao IntencaoCliente
	if err := json.Unmarshal([]byte(resposta), &intencao); err != nil {
		// Tenta extrair JSON do texto
		if jsonStr := extractJSONFromText(resposta); jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &intencao); err != nil {
				return nil, fmt.Errorf("erro ao parsear resposta da IA: %v\nResposta: %s", err, resposta)
			}
		} else {
			return nil, fmt.Errorf("erro ao parsear resposta da IA: %v\nResposta: %s", err, resposta)
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
			// Tenta encontrar o item no cardápio
			encontrou, preco := itemExisteNoCardapio(cardapio, item.Nome)
			if !encontrou {
				// Tenta encontrar similar
				similar := encontrarItemSimilar(cardapio, item.Nome)
				if similar != "" {
					intencao.Itens[i].Nome = similar
					log.Printf("[LLM] Item '%s' corrigido para '%s'", item.Nome, similar)
				} else {
					log.Printf("[LLM] Item '%s' não encontrado no cardápio", item.Nome)
				}
			} else {
				intencao.Itens[i].PrecoUnitario = preco
			}

			// Garante quantidade mínima
			if intencao.Itens[i].Quantidade <= 0 {
				intencao.Itens[i].Quantidade = 1
			}
		}
	}

	// Remove itens duplicados para adicionar
	if intencao.Acao == "adicionar" {
		intencao.Itens = mergeItens(intencao.Itens)
	}

	log.Printf("[LLM] Intenção detectada: %s, %d itens", intencao.Acao, len(intencao.Itens))

	return &intencao, nil
}
