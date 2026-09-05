package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	whatsappdto "github.com/rafapasa/mcp-server-openerp/internal/dto/whatsapp"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/require"
)

func TestWhatsAppClientSendButtons(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &WhatsAppClient{
		apiURL:      server.URL,
		phoneNumber: "phone-id",
		client:      server.Client(),
	}

	err := client.SendButtons(context.Background(), "5511999999999", "Escolha uma opção", []string{
		"Adicionar mais",
		"Finalizar pedido",
		"Limpar",
	})
	require.NoError(t, err)
	require.Equal(t, "interactive", payload["type"])

	interactive := payload["interactive"].(map[string]interface{})
	buttons := interactive["action"].(map[string]interface{})["buttons"].([]interface{})
	require.Equal(t, "reply", buttons[0].(map[string]interface{})["type"])
	require.Equal(t, "carrinho_adicionar", buttons[0].(map[string]interface{})["reply"].(map[string]interface{})["id"])
	require.Equal(t, "carrinho_finalizar", buttons[1].(map[string]interface{})["reply"].(map[string]interface{})["id"])
	require.Equal(t, "carrinho_limpar", buttons[2].(map[string]interface{})["reply"].(map[string]interface{})["id"])
}

func TestProcessorBuildMessageInputButton(t *testing.T) {
	processor := &Processor{}
	input := processor.buildMessageInput(context.Background(), whatsappdto.Message{
		Type: "interactive",
		Interactive: whatsappdto.Interactive{
			Type: "button",
			ButtonReply: whatsappdto.ButtonReply{
				ID: "carrinho_finalizar",
			},
		},
	})

	require.Equal(t, models.SourceText, input.Source)
	require.Equal(t, "carrinho_finalizar", input.Text)
}

func TestProcessorSendPaymentResponse(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		gotBody = payload["type"].(string)
		interactive := payload["interactive"].(map[string]interface{})
		buttons := interactive["action"].(map[string]interface{})["buttons"].([]interface{})
		require.Equal(t, "pagamento_1", buttons[0].(map[string]interface{})["reply"].(map[string]interface{})["id"])
		require.Equal(t, "Dinheiro", buttons[0].(map[string]interface{})["reply"].(map[string]interface{})["title"])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	processor := &Processor{
		whatsApp: &WhatsAppClient{
			apiURL:      server.URL,
			phoneNumber: "phone-id",
			client:      server.Client(),
		},
	}
	processor.sendPaymentResponse(context.Background(), "5511999999999", "💳 **COMO VAI PAGAR?**\n\n*1* - Dinheiro\n*2* - Pix")
	require.Equal(t, "interactive", gotBody)
}
