// internal/webhook/handlers.go - CLEAN - só HTTP + DTO
package webhook

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/mcp-server-openerp/internal/cache"
	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	whatsApp        *WhatsAppClient
	tenantService   service.TenantServiceInterface
	clienteService  service.ClienteServiceInterface
	carrinhoService service.CarrinhoServiceInterface
	cacheLayer      *cache.Cache
	cfg             *config.Config
}

func NewWebhookHandler(
	whatsApp *WhatsAppClient,
	tenantService service.TenantServiceInterface,
	clienteService service.ClienteServiceInterface,
	carrinhoService service.CarrinhoServiceInterface,
	cacheLayer *cache.Cache,
	cfg *config.Config,
) *WebhookHandler {
	return &WebhookHandler{
		whatsApp:        whatsApp,
		tenantService:   tenantService,
		clienteService:  clienteService,
		carrinhoService: carrinhoService,
		cacheLayer:      cacheLayer,
		cfg:             cfg,
	}
}

// WebhookRequest - só o necessário
type WebhookRequest struct {
	Object string `json:"object"`
	Entry  []struct {
		Changes []struct {
			Value struct {
				Metadata struct {
					PhoneNumberID      string `json:"phone_number_id"`
					DisplayPhoneNumber string `json:"display_phone_number"`
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
						ID string `json:"id"`
					} `json:"audio"`
					Voice struct {
						ID string `json:"id"`
					} `json:"voice"`
					Image struct {
						ID      string `json:"id"`
						Caption string `json:"caption"`
					} `json:"image"`
				} `json:"messages"`
				Statuses []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"statuses"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// 1,2,3,5 - verifica HMAC, valida JSON, responde HTTP
func (h *WebhookHandler) HandleWebhookFiber(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// VERIFICAÇÃO GET - hub.challenge
	if c.Method() == fiber.MethodGet {
		mode := c.Query("hub.mode")
		token := c.Query("hub.verify_token")
		challenge := c.Query("hub.challenge")
		if mode == "subscribe" && token == h.cfg.WhatsAppVerifyToken {
			return c.Status(200).SendString(challenge)
		}
		return c.Status(403).SendString("forbidden")
	}

	// POST - verifica HMAC
	ok, err := VerifyWebhookHandlerFiber(c, *h.cfg)
	if !ok {
		logger.Warn(ctx, "HMAC falhou", zap.Error(err), zap.String("ip", c.IP()))
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	body := c.Body()
	if len(body) == 0 {
		return c.Status(200).JSON(fiber.Map{"success": true})
	}
	// ignora status callbacks
	if !strings.Contains(string(body), "messages") {
		return c.Status(200).JSON(fiber.Map{"success": true})
	}

	var req WebhookRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error(ctx, "JSON inválido", zap.Error(err))
		return c.Status(200).JSON(fiber.Map{"success": true}) // 200 pra não retentar
	}

	// processa async mas responde 200 rápido
	// pra não travar, processa em goroutine com timeout
	go h.processEntries(context.Background(), req)

	return c.Status(200).JSON(fiber.Map{"success": true})
}

func (h *WebhookHandler) processEntries(ctx context.Context, req WebhookRequest) {
	for _, entry := range req.Entry {
		for _, change := range entry.Changes {
			phoneID := change.Value.Metadata.PhoneNumberID
			displayPhone := change.Value.Metadata.DisplayPhoneNumber

			tenantID, err := h.getTenantID(ctx, displayPhone, phoneID)
			if err != nil || tenantID == 0 {
				logger.Error(ctx, "tenant não resolvido", zap.String("display", displayPhone), zap.String("phoneID", phoneID))
				continue
			}

			for _, msg := range change.Value.Messages {
				// 4 - DEDUPLICAÇÃO - message_id no cache 5min
				if h.isDuplicate(ctx, msg.ID) {
					logger.Info(ctx, "Mensagem duplicada ignorada", zap.String("msg_id", msg.ID))
					continue
				}

				// resolve cliente
				contactName := ""
				if len(change.Value.Contacts) > 0 {
					contactName = change.Value.Contacts[0].Profile.Name
				}
				cliente, err := h.getOrCreateCliente(ctx, msg.From, contactName, tenantID)
				if err != nil {
					logger.Error(ctx, "Erro cliente", zap.Error(err))
					continue
				}

				// 4 - CONVERTE PARA DTO
				input := h.buildMessageInput(ctx, msg)

				// 4 - CHAMA SERVICE - 1 chamada só
				resposta, err := h.carrinhoService.ProcessarMensagem(ctx, cliente.ID, tenantID, input)
				if err != nil {
					logger.Error(ctx, "Erro ProcessarMensagem", zap.Error(err))
					resposta = "⚠️ Tive um problema técnico, tenta de novo por favor."
				}

				if resposta != "" {
					_ = h.whatsApp.SendMessage(msg.From, resposta)
				}
			}
		}
	}
}

// Converte WhatsApp message -> dto.MessageInput
func (h *WebhookHandler) buildMessageInput(ctx context.Context, msg struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      struct {
		Body string `json:"body"`
	} `json:"text"`
	Audio struct {
		ID string `json:"id"`
	} `json:"audio"`
	Voice struct {
		ID string `json:"id"`
	} `json:"voice"`
	Image struct {
		ID      string `json:"id"`
		Caption string `json:"caption"`
	} `json:"image"`
},
) dto.MessageInput {
	switch msg.Type {
	case "audio", "voice":
		mediaID := msg.Audio.ID
		if mediaID == "" {
			mediaID = msg.Voice.ID
		}
		if mediaID != "" {
			if data, err := h.whatsApp.DownloadMedia(ctx, mediaID); err == nil {
				return dto.MessageInput{
					Source: models.SourceAudio,
					Audio:  data,
				}
			}
		}
		return dto.MessageInput{Source: models.SourceAudio}

	case "image":
		caption := msg.Image.Caption
		if msg.Image.ID != "" {
			if data, err := h.whatsApp.DownloadMedia(ctx, msg.Image.ID); err == nil {
				return dto.MessageInput{
					Source: models.SourceImage,
					Image:  data,
					Text:   caption,
				}
			}
		}
		return dto.MessageInput{Source: models.SourceImage, Text: caption, Image: nil}

	default: // text, button, etc
		return dto.MessageInput{
			Source: models.SourceText,
			Text:   msg.Text.Body,
		}
	}
}

func (h *WebhookHandler) isDuplicate(ctx context.Context, messageID string) bool {
	if messageID == "" {
		return false
	}
	key := "wamid:dedupe:" + messageID
	// GetOrSet retorna true se já existe
	if h.cacheLayer == nil {
		return false
	}
	// tenta set com NX 5min - se já existe, é duplicado
	exists, _ := h.cacheLayer.Exists(key)
	if exists {
		return true
	}
	_ = h.cacheLayer.Set(key, "1", 5*time.Minute)
	return false
}

func (h *WebhookHandler) getOrCreateCliente(ctx context.Context, telefone, nome string, tenantID uint) (*dto.ClienteDTO, error) {
	// normaliza telefone
	telefone = strings.ReplaceAll(telefone, "+", "")
	telefone = strings.ReplaceAll(telefone, " ", "")
	cliente, err := h.clienteService.FindByTelefone(ctx, telefone, tenantID)
	if err != nil {
		return nil, err
	}
	if cliente == nil {
		cliente, err = h.clienteService.Create(ctx, &dto.CriarClienteRequest{
			TenantID: tenantID,
			Telefone: telefone,
			Nome:     nome,
		})
		if err != nil {
			return nil, err
		}
	}
	return cliente, err
}

func (h *WebhookHandler) getTenantID(_ context.Context, phoneNumber, phoneNumberID string) (uint, error) {
	// TODO: mover pra tenantService.ResolveByPhone
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
	return 0, fmt.Errorf("tenant não encontrado")
}
