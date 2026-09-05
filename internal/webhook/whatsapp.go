// internal/webhook/whatsapp.go - CLEAN com ctx
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type WhatsAppClient struct {
	apiURL      string
	accessToken string
	phoneNumber string
	client      *http.Client
}

func NewWhatsAppClient() *WhatsAppClient {
	return &WhatsAppClient{
		apiURL:      os.Getenv("WHATSAPP_API_URL"), // ex: https://graph.facebook.com/v20.0
		accessToken: os.Getenv("WHATSAPP_ACCESS_TOKEN"),
		phoneNumber: os.Getenv("WHATSAPP_PHONE_NUMBER"),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SendMessage sends a WhatsApp message to the specified recipient.
func (w *WhatsAppClient) SendMessage(to, message string) error {
	return w.SendMessageCtx(context.Background(), to, message)
}

// SendMessageCtx sends a WhatsApp message using the provided context.
func (w *WhatsAppClient) SendMessageCtx(ctx context.Context, to, message string) error {
	logger.Debug(
		ctx, "Enviando mensagem WhatsApp",
		zap.String("to", to),
		zap.Int("message_size", len(message)),
	)
	url := fmt.Sprintf("%s/%s/messages", w.apiURL, w.phoneNumber)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": message,
		},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+w.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		logger.Error(ctx, err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro ao enviar mensagem: %d - %s", resp.StatusCode, string(body))
	}
	return nil
}

func (w *WhatsAppClient) SendButtons(ctx context.Context, to, body string, labels []string) error {
	if len(labels) == 0 || len(labels) > 3 {
		return fmt.Errorf("quantidade de botões inválida: %d", len(labels))
	}
	buttons := make([]map[string]string, len(labels))
	for i, label := range labels {
		buttons[i] = map[string]string{"type": "reply", "id": buttonID(label), "title": label}
	}
	return w.sendButtonsPayload(ctx, to, body, buttons)
}

func (w *WhatsAppClient) sendButtonsWithIDs(ctx context.Context, to, body string, ids, labels []string) error {
	if len(ids) == 0 || len(ids) != len(labels) || len(ids) > 3 {
		return fmt.Errorf("quantidade de botões inválida")
	}
	buttons := make([]map[string]string, len(labels))
	for i := range labels {
		buttons[i] = map[string]string{"type": "reply", "id": ids[i], "title": labels[i]}
	}
	return w.sendButtonsPayload(ctx, to, body, buttons)
}

func (w *WhatsAppClient) sendButtonsPayload(ctx context.Context, to, body string, buttons []map[string]string) error {
	replies := make([]map[string]interface{}, 0, len(buttons))
	for _, button := range buttons {
		replies = append(replies, map[string]interface{}{
			"type": "reply",
			"reply": map[string]string{
				"id":    button["id"],
				"title": button["title"],
			},
		})
	}
	return w.sendJSON(ctx, map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "button",
			"body": map[string]string{"text": body},
			"action": map[string]interface{}{
				"buttons": replies,
			},
		},
	})
}

func buttonID(label string) string {
	switch label {
	case "Adicionar mais":
		return "carrinho_adicionar"
	case "Finalizar pedido":
		return "carrinho_finalizar"
	case "Limpar":
		return "carrinho_limpar"
	default:
		return label
	}
}

func (w *WhatsAppClient) sendJSON(ctx context.Context, payload interface{}) error {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/messages", w.apiURL, w.phoneNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+w.accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro ao enviar mensagem: %d - %s", resp.StatusCode, string(body))
	}
	return nil
}

// DownloadMedia - agora com ctx, usado só pra converter pra DTO
func (w *WhatsAppClient) DownloadMedia(ctx context.Context, mediaID string) ([]byte, error) {
	// 1. pega URL real da mídia
	url := fmt.Sprintf("%s/%s", w.apiURL, mediaID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("falha ao resolver media url %d: %s", resp.StatusCode, string(bb))
	}

	var meta struct {
		URL      string `json:"url"`
		MimeType string `json:"mime_type"`
	}
	bb, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bb, &meta); err != nil {
		return nil, fmt.Errorf("parse meta media: %w - %s", err, string(bb))
	}
	if meta.URL == "" {
		return nil, fmt.Errorf("sem url de mídia: %s", string(bb))
	}

	// 2. baixa binário
	req2, err := http.NewRequestWithContext(ctx, "GET", meta.URL, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Authorization", "Bearer "+w.accessToken)

	resp2, err := w.client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		b, _ := io.ReadAll(resp2.Body)
		return nil, fmt.Errorf("falha download binário %d: %s", resp2.StatusCode, string(b))
	}

	return io.ReadAll(resp2.Body)
}
