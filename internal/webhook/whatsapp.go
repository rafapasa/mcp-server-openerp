// internal/webhook/whatsapp.go
package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
