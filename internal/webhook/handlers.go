package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/rafapasa/mcp-server-openerp/internal/server"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// WebhookHandler gerencia as requisições do webhook
type WebhookHandler struct {
	mcpServer *server.MCPServer
	whatsApp  *WhatsAppClient
}

// NewWebhookHandler cria um novo handler
func NewWebhookHandler(mcpServer *server.MCPServer, whatsApp *WhatsAppClient) *WebhookHandler {
	return &WebhookHandler{
		mcpServer: mcpServer,
		whatsApp:  whatsApp,
	}
}

// WebhookRequest representa a estrutura da requisição do WhatsApp
type WebhookRequest struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text"`
					Type string `json:"type"`
				} `json:"messages"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

// WebhookResponse representa a resposta do webhook
type WebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// HandleWebhook processa as mensagens recebidas do WhatsApp
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Verifica método
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verifica assinatura (implementar com a lib do WhatsApp)
	// ... verifySignature(r) ...

	// Lê o body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Webhook] Erro ao ler body: %v", err)
		http.Error(w, "Erro ao ler body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Body: %s", body)
	// Parse da requisição
	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[Webhook] Erro ao parsear JSON: %v", err)
		http.Error(w, "Erro ao parsear JSON", http.StatusBadRequest)
		return
	}

	// Processa cada mensagem
	for _, entry := range req.Entry {
		for _, change := range entry.Changes {
			for _, message := range change.Value.Messages {
				// Extrai dados do cliente
				clienteID := message.From
				clienteNome := ""
				for _, contact := range change.Value.Contacts {
					if contact.WaID == clienteID {
						clienteNome = contact.Profile.Name
						break
					}
				}

				// O tenant_id precisa ser identificado de alguma forma
				// Pode vir do número de telefone do estabelecimento
				tenantID := h.getTenantID(change.Value.Metadata.PhoneNumberID)

				// Processa a mensagem
				go h.processMessage(clienteID, clienteNome, tenantID, message.Text.Body)
			}
		}
	}

	// Responde 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(WebhookResponse{
		Success: true,
		Message: "Mensagem processada com sucesso",
	})
}

// internal/webhook/handlers.go
func (h *WebhookHandler) HandleVerifyWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Pega os parâmetros da query string
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	// Verifica o token
	expectedToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")
	if mode == "subscribe" && token == expectedToken {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge)) // ← Retorna o challenge
		return
	}

	http.Error(w, "Verificação falhou", http.StatusForbidden)
}

// processMessage processa uma mensagem do WhatsApp
func (h *WebhookHandler) processMessage(clienteID, clienteNome, tenantID, mensagem string) {
	log.Printf("[Webhook] Mensagem de %s (%s): %s", clienteID, tenantID, mensagem)

	// Aqui chamamos a tool do MCP
	// Como não temos acesso direto à tool, vamos usar o service
	// ou criar uma função auxiliar

	// Opção 1: Criar um método no MCPServer para processar mensagens
	// Opção 2: Extrair a lógica da tool para um service comum
	// Opção 3: Chamar a tool via HTTP (se o MCP Server estiver rodando)

	// Implementação temporária - vamos processar diretamente
	// Isso precisa ser refinado
	ctx := context.Background()

	// Busca cardápio
	cardapio, err := h.mcpServer.GetCardapio(tenantID)
	if err != nil {
		log.Printf("[Webhook] Erro ao buscar cardápio: %v", err)
		return
	}

	// Extrai intenção
	intencao, err := h.mcpServer.ExtractIntent(mensagem, cardapio)
	if err != nil {
		log.Printf("[Webhook] Erro ao extrair intenção: %v", err)
		return
	}

	// Processa baseado na intenção
	var resposta string
	switch intencao.Acao {
	case "adicionar", "add":
		// Adiciona ao carrinho
		for _, item := range intencao.Itens {
			carrinhoItem := service.ItemCarrinho{
				Nome:       item.Nome,
				Quantidade: item.Quantidade,
				Observacao: item.Observacao,
				Preco:      item.PrecoUnitario,
			}
			h.mcpServer.AdicionarItemCarrinho(clienteID, tenantID, carrinhoItem)
		}
		resposta = h.mcpServer.FormatarResumoCarrinho(clienteID, tenantID)

	case "remover", "remove":
		for _, item := range intencao.Itens {
			h.mcpServer.RemoverItemCarrinho(clienteID, tenantID, item.Nome, item.Quantidade)
		}
		resposta = h.mcpServer.FormatarResumoCarrinho(clienteID, tenantID)

	case "finalizar", "confirmar":
		pedido, err := h.mcpServer.FinalizarCarrinho(ctx, clienteID, tenantID, clienteNome)
		if err != nil {
			resposta = fmt.Sprintf("❌ Erro ao finalizar pedido: %v", err)
		} else {
			resposta = h.mcpServer.FormatarRespostaPedido(pedido)
		}

	case "limpar", "clear":
		h.mcpServer.LimparCarrinho(clienteID, tenantID)
		resposta = "🗑️ Carrinho limpo com sucesso!"

	default:
		resposta = h.mcpServer.FormatarResumoCarrinho(clienteID, tenantID)
	}

	// Envia resposta
	if err := h.whatsApp.SendMessage(clienteID, resposta); err != nil {
		log.Printf("[Webhook] Erro ao enviar mensagem: %v", err)
	}
}

// getTenantID identifica o tenant baseado no número de telefone
func (h *WebhookHandler) getTenantID(phoneNumberID string) string {
	// Mapeamento entre número do WhatsApp Business e tenant
	// Pode ser feito via .env, banco de dados ou configuração
	// Exemplo simples:
	mapping := map[string]string{
		"123456789": "1", // PhoneNumberID do restaurante 1
		"987654321": "2", // PhoneNumberID do mercado 2
	}
	if tenantID, ok := mapping[phoneNumberID]; ok {
		return tenantID
	}
	// Fallback: usar o próprio phoneNumberID como tenant
	return phoneNumberID
}
