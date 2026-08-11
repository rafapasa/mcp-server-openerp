package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

// ============================================
// FUNÇÕES DE FORMATAÇÃO DE PROMPT
// ============================================

// formatarCardapioParaPrompt formata o cardápio para o prompt da IA
func formatarCardapioParaPrompt(cardapio []dto.ProdutoItem) string {
	var sb strings.Builder
	sb.WriteString("CARDÁPIO:\n")

	categoriaAtual := ""
	for _, item := range cardapio {
		if item.Categoria != categoriaAtual {
			categoriaAtual = item.Categoria
			sb.WriteString(fmt.Sprintf("\n--- %s ---\n", item.Categoria))
		}
		sb.WriteString(fmt.Sprintf("- %s: R$ %.2f", item.Nome, item.Preco))
		if item.Descricao != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", item.Descricao))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ============================================
// FUNÇÕES DE LIMPEZA DE RESPOSTA
// ============================================

// cleanJSONResponse limpa a resposta da IA removendo markdown e espaços extras
func cleanJSONResponse(resposta string) string {
	resposta = strings.TrimSpace(resposta)
	resposta = strings.TrimPrefix(resposta, "```json")
	resposta = strings.TrimPrefix(resposta, "```JSON")
	resposta = strings.TrimPrefix(resposta, "```")
	resposta = strings.TrimSuffix(resposta, "```")
	resposta = strings.TrimSpace(resposta)
	return resposta
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

// ============================================
// FUNÇÕES DE VALIDAÇÃO DE CARDÁPIO
// ============================================

// itemExisteNoCardapio verifica se um item existe e retorna seu preço
func itemExisteNoCardapio(cardapio []dto.ProdutoItem, nome string) (bool, float64) {
	nomeLower := strings.ToLower(strings.TrimSpace(nome))

	for _, item := range cardapio {
		if strings.ToLower(item.Nome) == nomeLower {
			return true, item.Preco
		}
		if strings.Contains(strings.ToLower(item.Nome), nomeLower) ||
			strings.Contains(nomeLower, strings.ToLower(item.Nome)) {
			return true, item.Preco
		}
	}

	return false, 0
}

// encontrarItemSimilar tenta encontrar um item similar no cardápio
func encontrarItemSimilar(cardapio []dto.ProdutoItem, nome string) string {
	nomeLower := strings.ToLower(strings.TrimSpace(nome))
	bestMatch := ""
	bestScore := 0

	for _, item := range cardapio {
		itemLower := strings.ToLower(item.Nome)
		score := similarityScore(nomeLower, itemLower)
		if score > bestScore {
			bestScore = score
			bestMatch = item.Nome
		}
	}

	if bestScore > 3 {
		return bestMatch
	}
	return ""
}

// similarityScore calcula uma pontuação de similaridade simples
func similarityScore(a, b string) int {
	score := 0
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	for _, wordA := range wordsA {
		for _, wordB := range wordsB {
			if len(wordA) > 3 && len(wordB) > 3 {
				if strings.Contains(wordA, wordB) || strings.Contains(wordB, wordA) {
					score += 2
				}
			}
		}
	}

	return score
}

// ============================================
// FUNÇÕES DE NORMALIZAÇÃO
// ============================================

// normalizeAcao normaliza a ação para um dos valores padrão
func normalizeAcao(acao string) string {
	acao = strings.ToLower(strings.TrimSpace(acao))

	switch acao {
	case "add", "adicionar", "adiciona", "incluir", "mais", "+":
		return "adicionar"
	case "remove", "remover", "remova", "tirar", "excluir", "-":
		return "remover"
	case "finalizar", "confirmar", "fechar", "confirm", "finish", "pagar", "terminar":
		return "finalizar"
	case "limpar", "clear", "apagar", "esvaziar", "reset":
		return "limpar"
	default:
		return "visualizar"
	}
}

// mergeItens mescla itens com o mesmo nome (soma quantidades)
func mergeItens(itens []dto.ItemPedidoInput) []dto.ItemPedidoInput {
	if len(itens) == 0 {
		return itens
	}

	merged := make(map[string]*dto.ItemPedidoInput)
	for _, item := range itens {
		if existing, ok := merged[item.Nome]; ok {
			existing.Quantidade += item.Quantidade
			if item.Observacao != "" && existing.Observacao == "" {
				existing.Observacao = item.Observacao
			}
		} else {
			merged[item.Nome] = &dto.ItemPedidoInput{
				Nome:          item.Nome,
				Quantidade:    item.Quantidade,
				Observacao:    item.Observacao,
				PrecoUnitario: item.PrecoUnitario,
			}
		}
	}

	result := make([]dto.ItemPedidoInput, 0, len(merged))
	for _, item := range merged {
		result = append(result, *item)
	}
	return result
}

// CorrigirNomes corrige nomes de produtos não encontrados
func CorrigirNomes(nomesNaoEncontrados []string, produtosEncontrados map[string]dto.ProdutoItem, generator func(string) (string, error)) ([]dto.ItemPedidoInput, error) {
	if len(nomesNaoEncontrados) == 0 || len(produtosEncontrados) == 0 {
		return []dto.ItemPedidoInput{}, nil
	}

	// Monta lista de produtos disponíveis
	var listaProdutos []string
	for nome := range produtosEncontrados {
		listaProdutos = append(listaProdutos, nome)
	}

	prompt := fmt.Sprintf(`
Você é um assistente especializado em corrigir nomes de produtos.

PRODUTOS DISPONÍVEIS:
%s

NOMES NÃO ENCONTRADOS:
%s

INSTRUÇÕES:
1. Para cada nome não encontrado, tente encontrar o produto mais similar na lista de disponíveis
2. Considere:
   - Typos (ex: "aroz" → "arroz")
   - Sinônimos (ex: "feijão" → "feijão carioca")
   - Abreviações (ex: "coca" → "coca-cola")
   - Erros de digitação comuns
3. Retorne APENAS o JSON com as correções

FORMATO DE RESPOSTA:
[
    {"nome_original": "aroz", "nome_corrigido": "arroz", "quantidade": 1},
    {"nome_original": "fejão", "nome_corrigido": "feijão carioca", "quantidade": 1}
]
`, strings.Join(listaProdutos, "\n"), strings.Join(nomesNaoEncontrados, "\n"))

	resposta, err := generator(prompt)
	if err != nil {
		return nil, err
	}

	// Parse da resposta
	var correcoes []struct {
		NomeOriginal  string `json:"nome_original"`
		NomeCorrigido string `json:"nome_corrigido"`
		Quantidade    int    `json:"quantidade"`
	}

	if err := json.Unmarshal([]byte(resposta), &correcoes); err != nil {
		return nil, err
	}

	// Converte para ItemPedidoInput
	var resultados []dto.ItemPedidoInput
	for _, corr := range correcoes {
		resultados = append(resultados, dto.ItemPedidoInput{
			Nome:       corr.NomeCorrigido,
			Quantidade: corr.Quantidade,
		})
	}

	return resultados, nil
}
