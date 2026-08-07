// internal/server/llm.go
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// OpenAILLM implementa LLMClient para OpenAI
type OpenAILLM struct {
	apiKey  string
	model   string
	baseURL string
}

// NewOpenAILLM cria um novo cliente OpenAI
func NewOpenAILLM() *OpenAILLM {
	return &OpenAILLM{
		apiKey:  os.Getenv("OPENAI_API_KEY"),
		model:   "gpt-4o-mini", // ou "gpt-4"
		baseURL: "https://api.openai.com/v1/chat/completions",
	}
}

// Generate envia um prompt para a OpenAI e retorna a resposta
func (llm *OpenAILLM) Generate(prompt string) (string, error) {
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

	req, err := http.NewRequest("POST", llm.baseURL, bytes.NewBuffer(jsonBody))
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

	if resp.StatusCode != 200 {
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

// extractOrderWithLLM chama a IA para extrair o pedido da mensagem
func (s *MCPServer) extractOrderWithLLM(mensagem string, cardapio []service.ProdutoItem) (*service.PedidoExtraido, error) {
	// 1. Monta o prompt
	cardapioStr := s.formatarCardapio(cardapio)

	prompt := fmt.Sprintf(`
Você é um assistente de atendimento de fast-food especializado em extrair pedidos.

%s

MENSAGEM DO CLIENTE:
"%s"

INSTRUÇÕES:
1. Extraia todos os itens do pedido mencionados na mensagem
2. Para cada item, identifique: nome, quantidade, e observações (ex: "sem cebola", "bem passado")
3. Separe itens principais de bebidas
4. Verifique se os itens existem no cardápio (use os nomes exatos ou os mais próximos)
5. Retorne APENAS o JSON, sem texto adicional

FORMATO DE RESPOSTA (JSON):
{
  "itens": [
    {"nome": "X-Bacon", "quantidade": 2, "observacao": "sem cebola"}
  ],
  "bebidas": [
    {"nome": "Coca-Cola", "quantidade": 1, "observacao": "tamanho médio"}
  ],
  "observacoes": "cliente pediu para entregar às 20h"
}`, cardapioStr, mensagem)

	// 2. Chama a IA
	resposta, err := s.llm.Generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar IA: %w", err)
	}

	// 3. Limpa a resposta (pode ter markdown)
	resposta = cleanJSONResponse(resposta)

	// 4. Parse do JSON
	var pedido service.PedidoExtraido
	if err := json.Unmarshal([]byte(resposta), &pedido); err != nil {
		// Tenta extrair JSON da resposta
		if jsonStr := extractJSONFromText(resposta); jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &pedido); err != nil {
				return nil, fmt.Errorf("erro ao parsear resposta da IA: %v\nResposta: %s", err, resposta)
			}
		} else {
			return nil, fmt.Errorf("erro ao parsear resposta da IA: %v\nResposta: %s", err, resposta)
		}
	}

	// 5. Valida e corrige os itens contra o cardápio
	for i, item := range pedido.Itens {
		existe, preco := s.itemExisteNoCardapio(cardapio, item.Nome)
		if !existe {
			similar := s.encontrarItemSimilar(cardapio, item.Nome)
			if similar != "" {
				pedido.Itens[i].Nome = similar
				log.Printf("[IA] Item '%s' corrigido para '%s'", item.Nome, similar)
			} else {
				log.Printf("[IA] Item '%s' não encontrado no cardápio", item.Nome)
			}
		} else {
			pedido.Itens[i].PrecoUnitario = preco
		}
	}

	// 6. Valida bebidas
	for i, item := range pedido.Bebidas {
		existe, preco := s.itemExisteNoCardapio(cardapio, item.Nome)
		if !existe {
			similar := s.encontrarItemSimilar(cardapio, item.Nome)
			if similar != "" {
				pedido.Bebidas[i].Nome = similar
			}
		} else {
			pedido.Bebidas[i].PrecoUnitario = preco
		}
	}

	log.Printf("[IA] Pedido extraído: %d itens, %d bebidas",
		len(pedido.Itens), len(pedido.Bebidas))

	return &pedido, nil
}

// cleanJSONResponse limpa a resposta da IA
func cleanJSONResponse(resposta string) string {
	// Remove markdown code blocks
	resposta = strings.ReplaceAll(resposta, "```json", "")
	resposta = strings.ReplaceAll(resposta, "```", "")
	return strings.TrimSpace(resposta)
}

// extractJSONFromText extrai JSON de um texto que pode ter outros caracteres
func extractJSONFromText(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || start > end {
		return ""
	}

	return text[start : end+1]
}
