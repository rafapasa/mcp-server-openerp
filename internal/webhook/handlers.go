// internal/webhook/handlers.go - FINAL 100% FIBER - CORRIGIDO
package webhook

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/cache"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/intent"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/media"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/server"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	mcpServer      *server.MCPServer
	whatsApp       *WhatsAppClient
	clienteService service.ClienteServiceInterface
	transcriber    *media.GroqTranscriber
	geminiLLM      llm.LLMClient
	deepseekLLM    llm.LLMClient
	cacheLayer     *cache.Cache
	cfg            *config.Config
}

func NewWebhookHandler(mcpServer *server.MCPServer, whatsApp *WhatsAppClient, clienteService service.ClienteServiceInterface, transcriber *media.GroqTranscriber, geminiLLM llm.LLMClient, deepseekLLM llm.LLMClient, cfg *config.Config) *WebhookHandler {
	return &WebhookHandler{
		mcpServer:      mcpServer,
		whatsApp:       whatsApp,
		clienteService: clienteService,
		transcriber:    transcriber,
		geminiLLM:      geminiLLM,
		deepseekLLM:    deepseekLLM,
		cfg:            cfg,
	}
}

// Injeção do cache - chamar em routes.go
func (h *WebhookHandler) SetCache(c *cache.Cache) {
	h.cacheLayer = c
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

// HandleWebhookFiber - POST /webhook
func (h *WebhookHandler) HandleWebhookFiber(c *fiber.Ctx) error {
	// Contexto Fiber - use Background com trace, UserContext() no Fiber é vazio
	ctx := context.Background()

	// 1. Valida assinatura PRIMEIRO
	ok, err := VerifyWebhookHandlerFiber(c, *h.cfg)
	if !ok {
		if !h.cfg.IsProduction() {
			logger.Warn(ctx, "Webhook não autorizado", zap.Error(err))
		}
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// 2. Fast check - ACK RÁPIDO pro Meta
	body := c.Body()
	if !strings.Contains(string(body), "\"messages\"") {
		logger.Debug(ctx, "Evento de status ignorado")
		return c.Status(200).JSON(WebhookResponse{Success: true})
	}

	// 3. Parse
	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Error(ctx, "Erro ao parsear JSON", zap.Error(err))
		return c.Status(200).JSON(WebhookResponse{Success: true}) // retorna 200 mesmo com erro pra Meta não reenviar
	}

	// 4. Responde 200 IMEDIATAMENTE e processa em background
	// Copia body pra goroutine não perder referência
	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)

	go h.processWebhookAsync(context.Background(), bodyCopy)

	return c.Status(200).JSON(WebhookResponse{Success: true, Message: "ok"})
}

// Processamento assíncrono - aqui pode demorar
func (h *WebhookHandler) processWebhookAsync(ctx context.Context, body []byte) {
	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Error(ctx, "Erro parse async", zap.Error(err))
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
				for _, contact := range change.Value.Contacts {
					if contact.WaID == userPhone {
						userName = contact.Profile.Name
						break
					}
				}
				tenantPhone := change.Value.Metadata.DisplayPhoneNumber
				tenantPhoneID := change.Value.Metadata.PhoneNumberID

				tenantID, err := h.getTenantID(ctx, tenantPhone, tenantPhoneID)
				if err != nil || tenantID == 0 {
					logger.Warn(ctx, "Tenant não encontrado", zap.String("tenantPhone", tenantPhone))
					continue
				}

				cliente, err := h.clienteService.BuscarOuCriarPorTelefone(ctx, tenantID, userPhone, userName)
				if err != nil {
					logger.Error(ctx, "Erro ao processar cliente", zap.Error(err))
					continue
				}

				// FAST PATH
				if message.Type == "text" {
					raw := strings.TrimSpace(message.Text.Body)

					var lastGreeting time.Time
					if h.cacheLayer != nil {
						if lastGreetStr, _ := h.cacheLayer.Get(ctx, fmt.Sprintf("last_greet:%d", cliente.ID)); lastGreetStr != "" {
							lastGreeting, _ = time.Parse(time.RFC3339, lastGreetStr)
						}
					}

					res := intent.ClassifyV2(raw, lastGreeting)

					switch res.Type {
					case intent.IntentGreeting:
						if h.cacheLayer != nil {
							h.cacheLayer.Set(ctx, fmt.Sprintf("last_greet:%d", cliente.ID), time.Now().Format(time.RFC3339), 10*time.Minute)
						}
						h.whatsApp.SendMessage(cliente.Telefone, intent.GreetingResponse(cliente.Nome, time.Now().Hour()))
						continue
					case intent.IntentGreetingWithAdd:
						if h.cacheLayer != nil {
							h.cacheLayer.Set(ctx, fmt.Sprintf("last_greet:%d", cliente.ID), time.Now().Format(time.RFC3339), 10*time.Minute)
						}
						h.whatsApp.SendMessage(cliente.Telefone, intent.GreetingResponse(cliente.Nome, time.Now().Hour()))
						h.processMessage(ctx, cliente, tenantID, res.CleanRest)
						continue
					case intent.IntentSmallTalk, intent.IntentThanks:
						h.whatsApp.SendMessage(cliente.Telefone, intent.SmallTalkResponse(raw))
						continue
					case intent.IntentViewCart:
						resposta := h.mcpServer.FormatarResumoCarrinho(ctx, cliente.ID, tenantID)
						h.whatsApp.SendMessage(cliente.Telefone, resposta)
						continue
					}
				}

				msgCopy := message
				h.processMessageWithMedia(ctx, cliente, tenantID, msgCopy)
			}
		}
	}
}

// HandleVerifyWebhookFiber - GET /webhook
func (h *WebhookHandler) HandleVerifyWebhookFiber(c *fiber.Ctx) error {
	ctx := context.Background()
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")
	expectedToken := strings.TrimSpace(os.Getenv("WHATSAPP_VERIFY_TOKEN"))
	if expectedToken == "" {
		expectedToken = strings.TrimSpace(h.cfg.WhatsAppVerifyToken)
	}

	if mode == "subscribe" && token == expectedToken && challenge != "" {
		c.Set("Content-Type", "text/plain")
		logger.Info(ctx, "Webhook verificado com sucesso", zap.String("mode", mode))
		return c.Status(200).SendString(challenge)
	}
	logger.Warn(ctx, "Verificação falhou", zap.String("mode", mode), zap.String("token", token))
	return c.Status(403).SendString("Verificação falhou")
}

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
		textoFinal = strings.TrimSpace(message.Text.Body)

	case "audio", "voice":
		mediaID := message.Audio.ID
		if message.Type == "voice" {
			mediaID = message.Voice.ID
		}
		audioBytes, err := h.whatsApp.DownloadMedia(mediaID)
		if err != nil {
			logger.Error(ctx, "Erro ao baixar audio", zap.Error(err))
			h.whatsApp.SendMessage(cliente.Telefone, "⚠ Não consegui baixar seu áudio. Pode enviar por texto?")
			return
		}
		transcribed, err := h.transcriber.Transcribe(ctx, audioBytes)
		if err != nil {
			logger.Error(ctx, "Erro transcrição Groq", zap.Error(err))
			h.whatsApp.SendMessage(cliente.Telefone, "⚠ Não entendi seu áudio. Pode digitar?")
			return
		}
		logger.Info(ctx, "Áudio transcrito", zap.String("texto", transcribed))
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
		promptVision := `Você é atendente. Analise imagem. Se receita: JSON {"tipo":"receita","texto_receita":"..."} Se produto/pedido: {"tipo":"pedido","descricao":"..."}. Só JSON.`
		visionResp, err := h.geminiLLM.GenerateWithImage(ctx, promptVision, b64, mime)
		if err != nil {
			textoFinal = message.Image.Caption
			if textoFinal == "" {
				textoFinal = "Cliente enviou imagem"
			}
		} else {
			var parsed map[string]interface{}
			if json.Unmarshal([]byte(visionResp), &parsed) == nil {
				if t, ok := parsed["texto_receita"]; ok {
					textoFinal = t.(string)
				} else if d, ok := parsed["descricao"]; ok {
					textoFinal = d.(string)
				} else {
					textoFinal = visionResp
				}
			} else {
				textoFinal = visionResp
			}
		}
		if message.Image.Caption != "" {
			textoFinal = textoFinal + " " + message.Image.Caption
		}

	case "document":
		docBytes, err := h.whatsApp.DownloadMedia(message.Document.ID)
		if err != nil {
			return
		}
		b64 := base64.StdEncoding.EncodeToString(docBytes)
		visionResp, _ := h.geminiLLM.GenerateWithImage(ctx, "Extraia texto desta receita em JSON {\"texto\":\"\"}", b64, message.Document.MimeType)
		textoFinal = visionResp

	default:
		logger.Warn(ctx, "Tipo não suportado", zap.String("type", message.Type))
		return
	}

	h.processMessage(ctx, cliente, tenantID, textoFinal)
}

func (h *WebhookHandler) processMessage(ctx context.Context, cliente *dto.ClienteDTO, tenantID uint, mensagem string) {
	logger.Info(ctx, "Processando mensagem",
		zap.Uint("cliente_id", cliente.ID),
		zap.String("telefone", cliente.Telefone),
		zap.Uint("tenant_id", tenantID),
		zap.String("mensagem", mensagem),
	)

	cardapio, err := cache.GetOrSet(h.cacheLayer, ctx, "cardapio:"+strconv.Itoa(int(tenantID)), 5*time.Minute, func() ([]dto.ProdutoItem, error) {
		c, err := h.mcpServer.GetCardapio(tenantID)
		if err != nil {
			return nil, err
		}
		return c, nil
	})
	if err != nil {
		logger.Error(ctx, "Erro ao buscar cardápio com cache", zap.Error(err))
		return
	}

	intencao, err := h.mcpServer.ExtractIntent(ctx, mensagem, cardapio)
	if err != nil {
		logger.Error(ctx, "Erro ao extrair intenção", zap.Error(err))
		h.whatsApp.SendMessage(cliente.Telefone, "⚠ Tive um problema técnico. Tente novamente.")
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
			resposta = "❌ Erro ao finalizar pedido."
		} else {
			resposta = h.mcpServer.FormatarRespostaPedido(ctx, pedido)
		}
	case "limpar", "clear":
		h.mcpServer.LimparCarrinho(ctx, cliente.ID, tenantID)
		resposta = "🗑 Carrinho limpo!"
	default:
		resposta = h.mcpServer.FormatarResumoCarrinho(ctx, cliente.ID, tenantID)
	}

	h.whatsApp.SendMessage(cliente.Telefone, resposta)
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
