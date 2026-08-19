// internal/webhook/whatsapp.go
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

// WhatsAppClient cliente para API do WhatsApp
type WhatsAppClient struct {
	apiURL      string
	accessToken string
	phoneNumber string
	client      *http.Client
}

// NewWhatsAppClient cria um novo cliente WhatsApp
func NewWhatsAppClient() *WhatsAppClient {
	return &WhatsAppClient{
		apiURL:      os.Getenv("WHATSAPP_API_URL"),
		accessToken: os.Getenv("WHATSAPP_ACCESS_TOKEN"),
		phoneNumber: os.Getenv("WHATSAPP_PHONE_NUMBER"),
		client:      &http.Client{},
	}
}

// SendMessage envia uma mensagem via WhatsApp
func (w *WhatsAppClient) SendMessage(to, message string) error {
	logger.Debug(context.Background(), "Enviando mensagem WhatsApp",
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

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
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

// NOVO - baixa mídia do WhatsApp Cloud API
func (w *WhatsAppClient) DownloadMedia(mediaID string) ([]byte, error) {
	// 1. pega URL real da mídia
	url := fmt.Sprintf("%s/%s", w.apiURL, mediaID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+w.accessToken)
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var meta struct {
		URL      string `json:"url"`
		MimeType string `json:"mime_type"`
	}
	bb, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bb, &meta)
	if meta.URL == "" {
		return nil, fmt.Errorf("sem url de mídia: %s", string(bb))
	}

	// 2. baixa binário
	req2, _ := http.NewRequest("GET", meta.URL, nil)
	req2.Header.Set("Authorization", "Bearer "+w.accessToken)
	resp2, err := w.client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()
	return io.ReadAll(resp2.Body)
}
