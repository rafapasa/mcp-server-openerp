// internal/webhook/processor.go
package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	whatsappdto "github.com/rafapasa/mcp-server-openerp/internal/dto/whatsapp"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/pkg/phone"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
	"go.uber.org/zap"
)

// Interfaces - só o que o processor precisa
type WhatsAppSender interface {
	SendMessage(to, text string) error
	DownloadMedia(ctx context.Context, mediaID string) ([]byte, error)
}

type Processor struct {
	whatsApp        WhatsAppSender
	tenantService   service.TenantServiceInterface
	clienteService  service.ClienteServiceInterface
	carrinhoService service.CarrinhoServiceInterface
	cacheLayer      database.RedisInterface
}

func NewProcessor(
	whatsApp *WhatsAppClient,
	tenantService service.TenantServiceInterface,
	clienteService service.ClienteServiceInterface,
	carrinhoService service.CarrinhoServiceInterface,
	cacheLayer database.RedisInterface,
) *Processor {
	return &Processor{
		whatsApp:        whatsApp,
		tenantService:   tenantService,
		clienteService:  clienteService,
		carrinhoService: carrinhoService,
		cacheLayer:      cacheLayer,
	}
}

func (p *Processor) Process(ctx context.Context, req whatsappdto.WebhookRequest) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(ctx, "panic no processor", zap.Any("recover", r))
		}
	}()

	for _, entry := range req.Entry {
		for _, change := range entry.Changes {
			tenantID, err := p.resolveTenant(ctx, change.Value.Metadata.DisplayPhoneNumber)
			if err != nil || tenantID == 0 {
				logger.Error(ctx, "tenant não resolvido",
					zap.String("display", change.Value.Metadata.DisplayPhoneNumber),
					zap.String("phoneID", change.Value.Metadata.PhoneNumberID),
					zap.Error(err))
				continue
			}

			for _, msg := range change.Value.Messages {
				if p.isDuplicate(ctx, msg.ID) {
					logger.Info(ctx, "Mensagem duplicada ignorada", zap.String("msg_id", msg.ID))
					continue
				}

				contactName := ""
				if len(change.Value.Contacts) > 0 {
					contactName = change.Value.Contacts[0].Profile.Name
				}

				cliente, err := p.getOrCreateCliente(ctx, msg.From, contactName, tenantID)
				if err != nil {
					logger.Error(ctx, "Erro ao resolver cliente", zap.Error(err), zap.String("telefone", msg.From))
					continue
				}
				if cliente == nil {
					logger.Error(ctx, "cliente nil após getOrCreate", zap.String("telefone", msg.From))
					continue
				}

				input := p.buildMessageInput(ctx, msg)

				resposta, err := p.carrinhoService.ProcessarMensagem(ctx, cliente.ID, tenantID, input)
				if err != nil {
					logger.Error(ctx, "Erro ProcessarMensagem", zap.Error(err), zap.Uint("cliente_id", cliente.ID))
					resposta = "⚠ Tive um problema técnico, tenta de novo por favor."
				}

				if resposta != "" {
					if err := p.whatsApp.SendMessage(msg.From, resposta); err != nil {
						logger.Error(ctx, "Erro ao enviar resposta WhatsApp", zap.Error(err), zap.String("to", msg.From))
					}
				}
			}
		}
	}
}

func (p *Processor) isDuplicate(ctx context.Context, messageID string) bool {
	if messageID == "" || p.cacheLayer == nil {
		return false
	}
	key := "wamid:dedupe:" + messageID
	set, err := p.cacheLayer.SetNXWithContext(ctx, key, "1", 5*time.Minute)
	if err != nil {
		logger.Warn(ctx, "falha ao checar deduplicação", zap.Error(err))
		return false
	}
	return !set
}

func (p *Processor) getOrCreateCliente(ctx context.Context, telefone, nome string, tenantID uint) (*dto.ClienteDTO, error) {
	normalized := phone.Normalize(telefone)
	if normalized == "" {
		return nil, fmt.Errorf("telefone vazio")
	}
	return p.clienteService.Create(ctx, &dto.CriarClienteRequest{
		TenantID: tenantID,
		Telefone: normalized,
		Nome:     nome,
	})
}

func (p *Processor) resolveTenant(ctx context.Context, displayPhone string) (uint, error) {
	clean := phone.Normalize(displayPhone)
	if clean == "" {
		return 0, fmt.Errorf("displayPhone vazio")
	}
	tenant, err := p.tenantService.GetByTelefone(ctx, clean)
	if err != nil {
		return 0, err
	}
	if tenant == nil || tenant.ID == 0 {
		return 0, fmt.Errorf("tenant não encontrado para %s", clean)
	}
	return tenant.ID, nil
}

func (p *Processor) buildMessageInput(ctx context.Context, msg whatsappdto.Message) dto.MessageInput {
	switch msg.Type {
	case "audio", "voice":
		mediaID := msg.Audio.ID
		if mediaID == "" {
			mediaID = msg.Voice.ID
		}
		if mediaID != "" {
			if data, err := p.whatsApp.DownloadMedia(ctx, mediaID); err == nil {
				return dto.MessageInput{Source: models.SourceAudio, Audio: data}
			}
			logger.Warn(ctx, "falha ao baixar audio", zap.String("media_id", mediaID))
		}
		return dto.MessageInput{Source: models.SourceAudio}

	case "image":
		caption := msg.Image.Caption
		if msg.Image.ID != "" {
			if data, err := p.whatsApp.DownloadMedia(ctx, msg.Image.ID); err == nil {
				return dto.MessageInput{Source: models.SourceImage, Image: data, Text: caption}
			}
			logger.Warn(ctx, "falha ao baixar imagem", zap.String("media_id", msg.Image.ID))
		}
		return dto.MessageInput{Source: models.SourceImage, Text: caption}

	default:
		return dto.MessageInput{Source: models.SourceText, Text: msg.Text.Body}
	}
}
