// internal/webhook/handlers.go
package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/security"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

// WebhookHandler gerencia as requisições do webhook
type WebhookHandler struct {
	mcpServer      *server.MCPServer
	whatsApp       *WhatsAppClient
	clienteService service.ClienteServiceInterface
}

// NewWebhookHandler cria um novo handler
func NewWebhookHandler(
	mcpServer *server.MCPServer,
	whatsApp *WhatsAppClient,
	clienteService service.ClienteServiceInterface,
) *WebhookHandler {
	return &WebhookHandler{
		mcpServer:      mcpServer,
		whatsApp:       whatsApp,
		clienteService: clienteService,
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
	ctx := r.Context()

	logger.Info(ctx, "Recebendo webhook do WhatsApp",
		zap.String("method", r.Method),
		zap.String("remote_addr", r.RemoteAddr),
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(ctx, "Erro ao ler body do webhook", zap.Error(err))
		http.Error(w, "Erro ao ler body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	logger.Debug(ctx, "Body do webhook recebido", zap.ByteString("body", body))

	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Error(ctx, "Erro ao parsear JSON do webhook", zap.Error(err))
		http.Error(w, "Erro ao parsear JSON", http.StatusBadRequest)
		return
	}

	for _, entry := range req.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				logger.Debug(ctx, "Evento ignorado (não é mensagem)",
					zap.String("field", change.Field),
				)
				continue
			}

			for _, message := range change.Value.Messages {
				userPhoneNumber := message.From
				userProfileName := ""
				for _, contact := range change.Value.Contacts {
					if contact.WaID == userPhoneNumber {
						userProfileName = contact.Profile.Name
						break
					}
				}

				tenantPhoneNumber := change.Value.Metadata.DisplayPhoneNumber
				tenantPhoneNumberID := change.Value.Metadata.PhoneNumberID

				if err := security.ValidatePhoneNumberID(userPhoneNumber); err != nil {
					logger.Warn(ctx, "Número do usuário inválido",
						zap.String("userPhoneNumber", userPhoneNumber),
						zap.Error(err))
					continue
				}

				if err := security.ValidatePhoneNumberID(tenantPhoneNumber); err != nil {
					logger.Warn(ctx, "Número do tenant inválido",
						zap.String("tenantPhoneNumber", tenantPhoneNumber),
						zap.Error(err))
					continue
				}

				tenantID, err := h.getTenantID(ctx, tenantPhoneNumber, tenantPhoneNumberID)
				if err != nil {
					logger.Warn(ctx, "Tenant não encontrado",
						zap.String("tenantPhoneNumber", tenantPhoneNumber),
						zap.Error(err))
					continue
				}

				cliente, err := h.clienteService.BuscarOuCriarPorTelefone(ctx, tenantID, userPhoneNumber, userProfileName)
				if err != nil {
					logger.Error(ctx, "Erro ao processar cliente",
						zap.Error(err),
						zap.String("userPhoneNumber", userPhoneNumber))
					continue
				}

				mensagem := message.Text.Body
				sanitizedMsg, err := security.SanitizeAndValidate(mensagem)
				if err != nil {
					logger.Warn(ctx, "Mensagem inválida",
						zap.Error(err),
						zap.String("mensagem", mensagem))
					continue
				}

				logger.Info(ctx, "Mensagem sanitizada",
					zap.String("original", mensagem),
					zap.String("sanitized", sanitizedMsg),
				)

				go h.processMessage(ctx, cliente, tenantID, sanitizedMsg, userProfileName)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(WebhookResponse{
		Success: true,
		Message: "Mensagem processada com sucesso",
	})

	logger.Info(ctx, "Webhook processado com sucesso",
		zap.Int("messages_count", len(req.Entry)))
}

// HandleVerifyWebhook verifica o webhook (Meta/WhatsApp)
func (h *WebhookHandler) HandleVerifyWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	expectedToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")
	if mode == "subscribe" && token == expectedToken {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
		return
	}

	logger.Warn(r.Context(), "Verificação do webhook falhou",
		zap.String("mode", mode),
		zap.String("token", token),
	)
	http.Error(w, "Verificação falhou", http.StatusForbidden)
}

// processMessage processa uma mensagem do WhatsApp
func (h *WebhookHandler) processMessage(
	ctx context.Context,
	cliente *dto.ClienteDTO,
	tenantID uint,
	mensagem string,
	userProfileName string,
) {
	logger.Info(ctx, "Processando mensagem",
		zap.Uint("cliente_id", cliente.ID),
		zap.String("cliente_nome", cliente.Nome),
		zap.String("cliente_telefone", cliente.Telefone),
		zap.Uint("tenant_id", tenantID),
		zap.String("mensagem", mensagem),
	)

	cardapio, err := h.mcpServer.GetCardapio(tenantID)
	if err != nil {
		logger.Error(ctx, "Erro ao buscar cardápio",
			zap.Error(err),
			zap.Uint("tenant_id", tenantID))
		return
	}

	intencao, err := h.mcpServer.ExtractIntent(ctx, mensagem, cardapio)
	if err != nil {
		logger.Error(ctx, "Erro ao extrair intenção",
			zap.Error(err),
			zap.String("mensagem", mensagem))
		// Mensagem amigável para o usuário
		resposta := "⚠️ Desculpe, estou com dificuldades técnicas. Por favor, tente novamente em alguns segundos."
		if err := h.whatsApp.SendMessage(cliente.Telefone, resposta); err != nil {
			logger.Error(ctx, "Erro ao enviar resposta WhatsApp", zap.Error(err))
		}
		return
	}

	var resposta string
	switch intencao.Acao {
	case "adicionar", "add":
		for _, item := range intencao.Itens {
			carrinhoItem := dto.ItemCarrinho{
				Nome:       item.Nome,
				Quantidade: item.Quantidade,
				Observacao: item.Observacao,
				Preco:      item.PrecoUnitario,
			}
			if err := h.mcpServer.AdicionarItemCarrinho(ctx, cliente.ID, tenantID, carrinhoItem); err != nil {
				logger.Error(ctx, "Erro ao adicionar item ao carrinho",
					zap.Error(err),
					zap.String("item", item.Nome),
				)
			}
		}
		resposta = h.mcpServer.FormatarResumoCarrinho(ctx, cliente.ID, tenantID)

	case "remover", "remove":
		for _, item := range intencao.Itens {
			if err := h.mcpServer.RemoverItemCarrinho(ctx, cliente.ID, tenantID, item.Nome, item.Quantidade); err != nil {
				logger.Error(ctx, "Erro ao remover item do carrinho",
					zap.Error(err),
					zap.String("item", item.Nome),
				)
			}
		}
		resposta = h.mcpServer.FormatarResumoCarrinho(ctx, cliente.ID, tenantID)

	case "finalizar", "confirmar":
		pedido, err := h.mcpServer.FinalizarCarrinho(ctx, cliente.ID, tenantID, cliente.Nome)
		if err != nil {
			logger.Error(ctx, "Erro ao finalizar pedido",
				zap.Error(err),
				zap.Uint("cliente_id", cliente.ID),
				zap.Uint("tenant_id", tenantID))
			resposta = "❌ Erro ao finalizar pedido. Por favor, tente novamente."
		} else {
			resposta = h.mcpServer.FormatarRespostaPedido(ctx, pedido)
		}

	case "limpar", "clear":
		if err := h.mcpServer.LimparCarrinho(ctx, cliente.ID, tenantID); err != nil {
			logger.Error(ctx, "Erro ao limpar carrinho", zap.Error(err))
			resposta = "❌ Erro ao limpar carrinho. Por favor, tente novamente."
		} else {
			resposta = "🗑️ Carrinho limpo com sucesso!"
		}

	default:
		resposta = h.mcpServer.FormatarResumoCarrinho(ctx, cliente.ID, tenantID)
	}

	if err := h.whatsApp.SendMessage(cliente.Telefone, resposta); err != nil {
		logger.Error(ctx, "Erro ao enviar resposta WhatsApp",
			zap.Error(err),
			zap.String("cliente_telefone", cliente.Telefone))
	}
}

// getTenantID busca o tenant pelo número de telefone
func (h *WebhookHandler) getTenantID(ctx context.Context, phoneNumber, phoneNumberID string) (uint, error) {
	// TODO: Implementar busca no banco de dados
	// return h.tenantService.FindByTelefone(ctx, phoneNumber)

	// Fallback: mapeamento estático (apenas para desenvolvimento)
	mapping := map[string]uint{
		"5511999999999": 1,
		"5511888888888": 2,
		"5511777777777": 3,
	}

	if id, ok := mapping[phoneNumber]; ok {
		return id, nil
	}

	if id, err := strconv.ParseUint(phoneNumberID, 10, 64); err == nil {
		return uint(id), nil
	}

	return 0, nil
}
