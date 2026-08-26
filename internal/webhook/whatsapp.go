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
		phoneNumber: os.Getenv("WHATSAPP_PHONE_NUMBER_ID"), // ID, não número
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (w *WhatsAppClient) SendMessage(to, message string) error {
	return w.SendMessageCtx(context.Background(), to, message)
}

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
