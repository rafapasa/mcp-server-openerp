package llm

import (
	"fmt"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

// formatarCardapioParaPrompt - ID + Nome + Preco, formato que a LLM entende sem chutar nome
func formatarCardapioParaPrompt(cardapio []dto.ProdutoItem) string {
	var sb strings.Builder
	for _, item := range cardapio {
		sb.WriteString(fmt.Sprintf("%d - %s - R$ %.2f\n", item.ID, item.Nome, item.Preco))
	}
	return sb.String()
}

func cleanJSONResponse(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```JSON")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func extractJSONFromText(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || start > end {
		return ""
	}
	return text[start : end+1]
}

func extractJSON(s string) string {
	s = cleanJSONResponse(s)
	if js := extractJSONFromText(s); js != "" {
		return js
	}
	return s
}
