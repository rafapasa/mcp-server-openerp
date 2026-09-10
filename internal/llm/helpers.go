package llm

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/etoolstec/gokit/apperror"
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

// mapLLMHTTPStatus mapeia status HTTP de providers externos para AppError.
// 429/503 → código preservado; demais 4xx → 400; 5xx → 500.
func mapLLMHTTPStatus(provider string, status int, body string) error {
	msg := fmt.Sprintf("%s status %d: %s", provider, status, truncate(body, 200))
	switch {
	case status == http.StatusTooManyRequests:
		return &apperror.AppError{Code: 429, Message: msg}
	case status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout:
		return &apperror.AppError{Code: 503, Message: msg}
	case status >= 500:
		return apperror.NewInternalError(msg, nil)
	case status >= 400:
		return apperror.NewBadRequestError(msg)
	default:
		return apperror.NewInternalError(msg, nil)
	}
}

func mapLLMNetErr(provider string, err error) error {
	if err == nil {
		return nil
	}
	if apperror.IsAppError(err) {
		return err
	}
	return apperror.NewInternalError(provider+" request failed", err)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
