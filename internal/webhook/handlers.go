// internal/webhook/handlers.go - FINAL 100% FIBER - Issue #8 FECHADO
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
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	whatsApp        *WhatsAppClient
	tenantService   service.TenantServiceInterface
	clienteService  service.ClienteServiceInterface
	cardapioService service.CardapioServiceInterface
	carrinhoService service.CarrinhoServiceInterface
	pedidoService   service.PedidoServiceInterface
	llmClient       *llm.UnifiedLLM
	cacheLayer      *cache.Cache
	cfg             *config.Config
}

func NewWebhookHandler(
	whatsApp *WhatsAppClient,
	tenantService service.TenantServiceInterface,
	clienteService service.ClienteServiceInterface,
	cardapioService service.CardapioServiceInterface,
	carrinhoService service.CarrinhoServiceInterface,
	pedidoService service.PedidoServiceInterface,
	llmClient *llm.UnifiedLLM,
	cfg *config.Config,
) *WebhookHandler {
	return &WebhookHandler{
		whatsApp:        whatsApp,
		tenantService:   tenantService,
		clienteService:  clienteService,
		cardapioService: cardapioService,
		carrinhoService: carrinhoService,
		pedidoService:   pedidoService,
		llmClient:       llmClient,
		cfg:             cfg,
	}
}

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

func (h *WebhookHandler) HandleWebhookFiber(c *fiber.Ctx) error {
	ctx := context.Background()
	logger.Info(
		ctx, "WEBHOOK ENTRADA",
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
		zap.String("ip", c.IP()),
		zap.String("headers", string(c.Request().Header.Header())),
		zap.String("body", string(c.Body())),
	)
	ok, err := VerifyWebhookHandlerFiber(c, *h.cfg)
	if !ok {
		logger.Warn(ctx, "Validação HMAC falhou", zap.Error(err))
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	body := c.Body()
	if !strings.Contains(string(body), "\"messages\"") {
		logger.Debug(ctx, "Evento de status ignorado")
		return c.Status(200).JSON(WebhookResponse{Success: true})
	}
	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Error(ctx, "Erro ao parsear JSON", zap.Error(err))
		return c.Status(200).JSON(WebhookResponse{Success: true})
	}
	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)
	go h.processWebhookAsync(context.Background(), bodyCopy)
	return c.Status(200).JSON(WebhookResponse{Success: true, Message: "ok"})
}

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
				// FAST PATH TEXT - Issue #8: View sem LLM
				if message.Type == "text" {
					raw := strings.TrimSpace(message.Text.Body)
					var lastGreeting time.Time
					if h.cacheLayer != nil {
						if lastGreetStr, _ := h.cacheLayer.Get(fmt.Sprintf("last_greet:%d", cliente.ID)); lastGreetStr != "" {
							lastGreeting, _ = time.Parse(time.RFC3339, lastGreetStr)
						}
					}
					res := intent.ClassifyV2(raw, lastGreeting)
					switch res.Type {
					case intent.IntentGreeting:
						if h.cacheLayer != nil {
							h.cacheLayer.Set(fmt.Sprintf("last_greet:%d", cliente.ID), time.Now().Format(time.RFC3339), 10*time.Minute)
						}
						logger.Info(ctx, "FAST-PATH Greeting - sem LLM", zap.String("msg", raw))
						h.whatsApp.SendMessage(cliente.Telefone, intent.GreetingResponse(cliente.Nome, time.Now().Hour()))
						continue
					case intent.IntentGreetingWithAdd:
						if h.cacheLayer != nil {
							h.cacheLayer.Set(fmt.Sprintf("last_greet:%d", cliente.ID), time.Now().Format(time.RFC3339), 10*time.Minute)
						}
						logger.Info(ctx, "FAST-PATH GreetingWithAdd", zap.String("cleanRest", res.CleanRest))
						h.whatsApp.SendMessage(cliente.Telefone, intent.GreetingResponse(cliente.Nome, time.Now().Hour()))
						h.processMessage(ctx, cliente, tenantID, res.CleanRest)
						continue
					case intent.IntentSmallTalk, intent.IntentThanks:
						logger.Info(ctx, "FAST-PATH SmallTalk/Thanks - sem LLM", zap.String("msg", raw))
						h.whatsApp.SendMessage(cliente.Telefone, intent.SmallTalkResponse(raw))
						continue
					case intent.IntentViewCart:
						logger.Info(ctx, "FAST-PATH ViewCart - sem LLM, GetOrSet 2m", zap.String("msg", raw), zap.Uint("cliente_id", cliente.ID))
						resposta, _ := h.carrinhoService.FormatResumoCarrinho(ctx, cliente.ID, tenantID)
						if resposta == "" {
							resposta = "🛒 Seu carrinho está vazio. Me diga o que quer adicionar!"
						}
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
},
) {
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
			h.whatsApp.SendMessage(cliente.Telefone, "⚠ Não consegui baixar seu áudio. Pode enviar por texto?")
			return
		}
		transcribed, err := h.llmClient.TranscribeAudio(ctx, audioBytes)
		if err != nil {
			h.whatsApp.SendMessage(cliente.Telefone, "⚠ Não entendi seu áudio. Pode digitar?")
			return
		}
		textoFinal = transcribed
	case "image", "document":
		var mediaID, mime, caption string
		if message.Type == "image" {
			mediaID = message.Image.ID
			mime = message.Image.MimeType
			caption = message.Image.Caption
		} else {
			mediaID = message.Document.ID
			mime = message.Document.MimeType
			caption = message.Document.Filename
		}
		if mime == "" {
			mime = "image/jpeg"
		}
		imgBytes, err := h.whatsApp.DownloadMedia(mediaID)
		if err != nil {
			logger.Error(ctx, "Erro baixar midia", zap.Error(err))
			return
		}
		tenant, _ := h.tenantService.GetByID(ctx, tenantID)
		cardapio, _ := h.cardapioService.GetCardapio(ctx, tenantID)
		cardapioStr := h.formatCardapioParaPrompt(cardapio)
		prompt := fmt.Sprintf(
			llm.PromptGenerateWithImage,
			tenant.Nome,
			tenant.Segmento,
			caption,
			cardapioStr,
			"Cliente: "+cliente.Nome,
		)
		b64 := base64.StdEncoding.EncodeToString(imgBytes)
		visionResp, err := h.llmClient.DescribeImage(ctx, []byte(b64), prompt)
		if err != nil {
			textoFinal = caption
			if textoFinal == "" {
				textoFinal = "Cliente enviou imagem"
			}
		} else {
			var parsed map[string]interface{}
			if json.Unmarshal([]byte(visionResp), &parsed) == nil {
				if t, ok := parsed["texto_receita"].(string); ok {
					textoFinal = t
				} else if d, ok := parsed["descricao"].(string); ok {
					textoFinal = d
				} else if txt, ok := parsed["texto"].(string); ok {
					textoFinal = txt
				} else {
					textoFinal = visionResp
				}
			} else {
				textoFinal = visionResp
			}
		}
		if caption != "" && !strings.Contains(textoFinal, caption) {
			textoFinal = textoFinal + " " + caption
		}
	default:
		return
	}
	h.processMessage(ctx, cliente, tenantID, textoFinal)
}

func (h *WebhookHandler) formatCardapioParaPrompt(produtos []dto.ProdutoItem) string {
	var sb strings.Builder
	for _, p := range produtos {
		sb.WriteString(fmt.Sprintf("- %s: R$ %.2f\n", p.Nome, p.Preco))
	}
	return sb.String()
}

func (h *WebhookHandler) processMessage(ctx context.Context, cliente *dto.ClienteDTO, tenantID uint, mensagem string) {
	logger.Info(
		ctx, "MSG RECEBIDA",
		zap.Uint("cliente_id", cliente.ID),
		zap.String("telefone", cliente.Telefone),
		zap.String("nome", cliente.Nome),
		zap.Uint("tenant", tenantID),
		zap.String("mensagem", mensagem),
	)

	// FIX Issue #8 - FAST-PATH também no processMessage (vem de áudio/imagem transcrita)
	viewCheck := intent.ClassifyV2(mensagem, time.Time{})
	if viewCheck.Type == intent.IntentViewCart {
		logger.Info(ctx, "FAST-PATH ViewCart no processMessage - sem LLM, GetOrSet 2m", zap.String("mensagem", mensagem))
		resposta, _ := h.carrinhoService.FormatResumoCarrinho(ctx, cliente.ID, tenantID)
		if resposta == "" {
			resposta = "🛒 Seu carrinho está vazio. Me diga o que quer adicionar!"
		}
		logger.Info(ctx, "Resposta enviada", zap.String("resposta", resposta))
		h.whatsApp.SendMessage(cliente.Telefone, resposta)
		return
	}

	cardapio, err := cache.GetOrSet(
		h.cacheLayer,
		ctx,
		"cardapio:"+strconv.Itoa(int(tenantID)),
		5*time.Minute,
		func() ([]dto.ProdutoItem, error) {
			c, err := h.cardapioService.GetCardapio(ctx, tenantID)
			if err != nil {
				return nil, err
			}
			return c, nil
		},
	)
	if err != nil {
		logger.Error(ctx, "Erro ao buscar cardápio com cache", zap.Error(err))
		return
	}

	intencao, err := h.llmClient.ExtractIntent(ctx, mensagem, cardapio)
	if err != nil {
		logger.Error(ctx, "Erro ao extrair intenção", zap.Error(err))
		h.whatsApp.SendMessage(cliente.Telefone, "⚠ Tive um problema técnico. Tente novamente.")
		return
	}
	logger.Info(ctx, "Intenção extraída", zap.Any("intencao", intencao))
	var resposta string
	switch intencao.Acao {
	case "adicionar", "add":
		for _, item := range intencao.Itens {
			h.carrinhoService.AdicionarItem(ctx, cliente.ID, tenantID, dto.ItemCarrinho{
				Nome: item.Nome, Quantidade: item.Quantidade, Observacao: item.Observacao, Preco: item.PrecoUnitario,
			})
		}
		resposta, _ = h.carrinhoService.FormatResumoCarrinho(ctx, cliente.ID, tenantID)
	case "remover", "remove":
		for _, item := range intencao.Itens {
			h.carrinhoService.RemoverItem(ctx, cliente.ID, tenantID, item.Nome, item.Quantidade)
		}
		resposta, _ = h.carrinhoService.FormatResumoCarrinho(ctx, cliente.ID, tenantID)
	case "finalizar", "confirmar":
		pedido, err := h.carrinhoService.FinalizarCarrinho(ctx, cliente.ID, tenantID, cliente.Nome)
		if err != nil {
			resposta = "❌ Erro ao finalizar pedido."
		} else {
			resposta = h.carrinhoService.FormatarPedidoConfirmado(pedido)
		}
	case "limpar", "clear":
		h.carrinhoService.LimparCarrinho(ctx, cliente.ID, tenantID)
		resposta = "🗑 Carrinho limpo!"
	default:
		resposta, _ = h.carrinhoService.FormatResumoCarrinho(ctx, cliente.ID, tenantID)
	}
	logger.Info(ctx, "Resposta enviada", zap.String("resposta", resposta))
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
