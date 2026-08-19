// internal/webhook/handlers.go
package webhook

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/media"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/security"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	mcpServer      *server.MCPServer
	whatsApp       *WhatsAppClient
	clienteService service.ClienteServiceInterface
	transcriber    *media.GroqTranscriber
	geminiLLM      llm.LLMClient // vision
	deepseekLLM    llm.LLMClient // texto - você já pode usar h.mcpServer.ExtractIntent que já está com deepseek via factory
}

func NewWebhookHandler(mcpServer *server.MCPServer, whatsApp *WhatsAppClient, clienteService service.ClienteServiceInterface, transcriber *media.GroqTranscriber, geminiLLM llm.LLMClient, deepseekLLM llm.LLMClient) *WebhookHandler {
	return &WebhookHandler{
		mcpServer:      mcpServer,
		whatsApp:       whatsApp,
		clienteService: clienteService,
		transcriber:    transcriber,
		geminiLLM:      geminiLLM,
		deepseekLLM:    deepseekLLM,
	}
}

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
					Type      string `json:"type"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text"`
					Audio struct {
						ID       string `json:"id"`
						MimeType string `json:"mime_type"`
					} `json:"audio"`
					Voice struct {
						ID       string `json:"id"`
						MimeType string `json:"mime_type"`
					} `json:"voice"`
					Image struct {
						ID       string `json:"id"`
						MimeType string `json:"mime_type"`
						Caption  string `json:"caption"`
					} `json:"image"`
					Document struct {
						ID       string `json:"id"`
						MimeType string `json:"mime_type"`
						Filename string `json:"filename"`
					} `json:"document"`
				} `json:"messages"`
				Statuses []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"statuses"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

type WebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(ctx, "Erro ao ler body do webhook", zap.Error(err))
		http.Error(w, "Erro ao ler body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if !strings.Contains(string(body), "\"messages\"") {
		logger.Debug(ctx, "Evento de status ignorado")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(WebhookResponse{Success: true})
		return
	}

	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Error(ctx, "Erro ao parsear JSON do webhook", zap.Error(err))
		http.Error(w, "Erro ao parsear JSON", http.StatusBadRequest)
		return
	}

	for _, entry := range req.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" || len(change.Value.Messages) == 0 {
				continue
			}
			for _, message := range change.Value.Messages {
				userPhone := message.From
				userName := ""
				for _, c := range change.Value.Contacts {
					if c.WaID == userPhone {
						userName = c.Profile.Name
						break
					}
				}
				tenantPhone := change.Value.Metadata.DisplayPhoneNumber
				tenantPhoneID := change.Value.Metadata.PhoneNumberID

				tenantID, err := h.getTenantID(ctx, tenantPhone, tenantPhoneID)
				if err != nil || tenantID == 0 {
					logger.Warn(ctx, "Tenant não encontrado", zap.String("tenantPhone", tenantPhone), zap.Error(err))
					continue
				}

				cliente, err := h.clienteService.BuscarOuCriarPorTelefone(ctx, tenantID, userPhone, userName)
				if err != nil {
					logger.Error(ctx, "Erro ao processar cliente", zap.Error(err))
					continue
				}

				// processa em background
				msgCopy := message
				go h.processMessageWithMedia(context.Background(), cliente, tenantID, msgCopy)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(WebhookResponse{Success: true, Message: "ok"})
}

func (h *WebhookHandler) HandleVerifyWebhook(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")
	expectedToken := strings.TrimSpace(os.Getenv("WHATSAPP_VERIFY_TOKEN"))
	if mode == "subscribe" && token == expectedToken && challenge != "" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
		logger.Info(r.Context(), "Webhook verificado com sucesso")
		return
	}
	logger.Warn(r.Context(), "Verificação do webhook falhou", zap.String("mode", mode))
	http.Error(w, "Verificação falhou", http.StatusForbidden)
}

// NOVO: roteador de mídia
func (h *WebhookHandler) processMessageWithMedia(ctx context.Context, cliente *dto.ClienteDTO, tenantID uint, message struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      struct {
		Body string `json:"body"`
	} `json:"text"`
	Audio struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
	} `json:"audio"`
	Voice struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
	} `json:"voice"`
	Image struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
		Caption  string `json:"caption"`
	} `json:"image"`
	Document struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
		Filename string `json:"filename"`
	} `json:"document"`
}) {
	var textoFinal string

	switch message.Type {
	case "text":
		sanitized, err := security.SanitizeAndValidate(message.Text.Body)
		if err != nil {
			logger.Warn(ctx, "Mensagem inválida", zap.Error(err))
			return
		}
		textoFinal = sanitized

	case "audio", "voice":
		mediaID := message.Audio.ID
		if message.Type == "voice" {
			mediaID = message.Voice.ID
		}
		audioBytes, err := h.whatsApp.DownloadMedia(mediaID)
		if err != nil {
			logger.Error(ctx, "Erro ao baixar audio", zap.Error(err))
			h.whatsApp.SendMessage(cliente.Telefone, "⚠️ Não consegui baixar seu áudio. Pode enviar por texto?")
			return
		}
		// GROQ WHISPER
		transcribed, err := h.transcriber.Transcribe(ctx, audioBytes)
		if err != nil {
			logger.Error(ctx, "Erro ao transcrever audio Groq", zap.Error(err))
			h.whatsApp.SendMessage(cliente.Telefone, "⚠️ Não consegui entender seu áudio. Pode digitar?")
			return
		}
		logger.Info(ctx, "Audio transcrito via Groq", zap.String("texto", transcribed))
		textoFinal = transcribed

	case "image":
		imgBytes, err := h.whatsApp.DownloadMedia(message.Image.ID)
		if err != nil {
			logger.Error(ctx, "Erro ao baixar imagem", zap.Error(err))
			return
		}
		b64 := base64.StdEncoding.EncodeToString(imgBytes)
		mime := message.Image.MimeType
		if mime == "" {
			mime = "image/jpeg"
		}

		// GEMINI VISION
		promptVision := `Você é um atendente de farmácia/restaurante. Analise a imagem.
		Se for receita médica: retorne JSON {"tipo":"receita","texto_receita":"...","medicamentos":[{"nome":"", "dosagem":""}]}
		Se for produto/cardápio/pizza/lanche: descreva o pedido que o cliente quer {"tipo":"pedido","descricao":"...","itens_sugeridos":[""]}.
		Retorne SÓ JSON.`

		visionResp, err := h.geminiLLM.GenerateWithImage(ctx, promptVision, b64, mime)
		if err != nil {
			logger.Error(ctx, "Erro Gemini Vision", zap.Error(err))
			textoFinal = message.Image.Caption
			if textoFinal == "" {
				textoFinal = "Cliente enviou uma imagem"
			}
		} else {
			logger.Info(ctx, "Gemini Vision extraiu", zap.String("resposta", visionResp))
			// Tenta parsear se for receita
			var parsed map[string]interface{}
			if json.Unmarshal([]byte(visionResp), &parsed) == nil {
				if parsed["tipo"] == "receita" {
					// Já busca medicamentos direto
					textoFinal = parsed["texto_receita"].(string)
					// Você pode aqui chamar serviço que busca no banco e responde
				} else {
					textoFinal = visionResp
					if desc, ok := parsed["descricao"]; ok {
						textoFinal = desc.(string)
					}
				}
			} else {
				textoFinal = visionResp
			}
		}
		// Adiciona caption se tiver
		if message.Image.Caption != "" {
			textoFinal = textoFinal + " " + message.Image.Caption
		}

	case "document":
		// Pode ser receita em PDF - baixa e manda pro Gemini também
		docBytes, err := h.whatsApp.DownloadMedia(message.Document.ID)
		if err != nil {
			return
		}
		b64 := base64.StdEncoding.EncodeToString(docBytes)
		mime := message.Document.MimeType
		visionResp, _ := h.geminiLLM.(interface {
			GenerateWithImage(context.Context, string, string, string) (string, error)
		}).GenerateWithImage(ctx, "Extraia texto desta receita/documento em JSON {\"texto\":\"\"}", b64, mime)
		textoFinal = visionResp

	default:
		logger.Warn(ctx, "Tipo de mensagem não suportado", zap.String("type", message.Type))
		return
	}

	// AGORA TUDO É TEXTO -> DeepSeek (via seu MCPServer que já usa factory)
	h.processMessage(ctx, cliente, tenantID, textoFinal)
}

func (h *WebhookHandler) processMessage(ctx context.Context, cliente *dto.ClienteDTO, tenantID uint, mensagem string) {
	logger.Info(ctx, "Processando mensagem",
		zap.Uint("cliente_id", cliente.ID),
		zap.String("telefone", cliente.Telefone),
		zap.Uint("tenant_id", tenantID),
		zap.String("mensagem", mensagem),
	)

	cardapio, err := h.mcpServer.GetCardapio(tenantID)
	if err != nil {
		logger.Error(ctx, "Erro ao buscar cardápio", zap.Error(err))
		return
	}

	intencao, err := h.mcpServer.ExtractIntent(ctx, mensagem, cardapio)
	if err != nil {
		logger.Error(ctx, "Erro ao extrair intenção", zap.Error(err))
		h.whatsApp.SendMessage(cliente.Telefone, "⚠ Desculpe, tive um problema técnico. Tente novamente em alguns segundos.")
		return
	}

	var resposta string
	switch intencao.Acao {
	case "adicionar", "add":
		for _, item := range intencao.Itens {
			h.mcpServer.AdicionarItemCarrinho(ctx, cliente.ID, tenantID, dto.ItemCarrinho{
				Nome: item.Nome, Quantidade: item.Quantidade, Observacao: item.Observacao, Preco: item.PrecoUnitario,
			})
		}
		resposta = h.mcpServer.FormatarResumoCarrinho(ctx, cliente.ID, tenantID)
	case "remover", "remove":
		for _, item := range intencao.Itens {
			h.mcpServer.RemoverItemCarrinho(ctx, cliente.ID, tenantID, item.Nome, item.Quantidade)
		}
		resposta = h.mcpServer.FormatarResumoCarrinho(ctx, cliente.ID, tenantID)
	case "finalizar", "confirmar":
		pedido, err := h.mcpServer.FinalizarCarrinho(ctx, cliente.ID, tenantID, cliente.Nome)
		if err != nil {
			resposta = "❌ Erro ao finalizar pedido. Tente novamente."
		} else {
			resposta = h.mcpServer.FormatarRespostaPedido(ctx, pedido)
		}
	case "limpar", "clear":
		h.mcpServer.LimparCarrinho(ctx, cliente.ID, tenantID)
		resposta = "🗑 Carrinho limpo!"
	default:
		resposta = h.mcpServer.FormatarResumoCarrinho(ctx, cliente.ID, tenantID)
	}

	if err := h.whatsApp.SendMessage(cliente.Telefone, resposta); err != nil {
		logger.Error(ctx, "Erro ao enviar resposta WhatsApp", zap.Error(err))
	}
}

func (h *WebhookHandler) getTenantID(_ context.Context, phoneNumber, phoneNumberID string) (uint, error) {
	mapping := map[string]uint{
		"554989014080":  1,
		"5511999999999": 2,
	}
	if id, ok := mapping[phoneNumber]; ok {
		return id, nil
	}
	clean := strings.ReplaceAll(strings.ReplaceAll(phoneNumber, "+", ""), " ", "")
	if id, ok := mapping[clean]; ok {
		return id, nil
	}
	if id, err := strconv.ParseUint(phoneNumberID, 10, 64); err == nil {
		return uint(id), nil
	}
	return 0, nil
}
