// internal/webhook/processor.go - FIX multi-tenant
package webhook

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	whatsappdto "github.com/rafapasa/mcp-server-openerp/internal/dto/whatsapp"
	"github.com/rafapasa/mcp-server-openerp/internal/intent"
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

type buttonSender interface {
	SendButtons(ctx context.Context, to, body string, labels []string) error
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
			// FIX: usa phone_number_id primeiro, fallback display_number
			tenantID, err := p.resolveTenant(ctx, change.Value.Metadata.PhoneNumberID, change.Value.Metadata.DisplayPhoneNumber)
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
				if p.isHumanHandoffActive(ctx, cliente.ID) && !intent.IsVoltarProBot(input.Text) {
					p.forwardToHuman(ctx, cliente.ID, tenantID, msg.From, input)
					if p.shouldPromptHandoffTimeout(ctx, cliente.ID) {
						if err := p.whatsApp.SendMessage(msg.From, "⏳ Seu atendimento humano ainda está aguardando resposta. Você prefere continuar aguardando ou *voltar pro bot*?"); err != nil {
							logger.Error(ctx, "Erro ao enviar alerta de handoff", zap.Error(err))
						}
					}
					continue
				}

				resposta, err := p.carrinhoService.ProcessarMensagem(ctx, cliente.ID, tenantID, input)
				if err != nil {
					logger.Error(ctx, "Erro ProcessarMensagem", zap.Error(err), zap.Uint("cliente_id", cliente.ID))
					resposta = "⚠ Tive um problema técnico, tenta de novo por favor."
				}

				if resposta != "" {
					if p.shouldSendAddressButtons(resposta) {
						p.sendAddressResponse(ctx, msg.From, resposta)
						continue
					}
					if p.shouldSendPaymentButtons(resposta) {
						p.sendPaymentResponse(ctx, msg.From, resposta)
						continue
					}
					if p.shouldSendCartButtons(input, resposta) {
						p.sendCartResponse(ctx, cliente.ID, tenantID, msg.From, resposta)
						continue
					}
					if err := p.whatsApp.SendMessage(msg.From, resposta); err != nil {
						logger.Error(ctx, "Erro ao enviar resposta WhatsApp", zap.Error(err), zap.String("to", msg.From))
					}

				}
			}
		}
	}
}

func (p *Processor) shouldSendCartButtons(input dto.MessageInput, response string) bool {
	return input.Source == models.SourceText &&
		strings.Contains(response, "🛒") &&
		strings.Contains(response, "Carrinho")
}

func (p *Processor) shouldSendPaymentButtons(response string) bool {
	return strings.Contains(response, "COMO VAI PAGAR?")
}

func (p *Processor) shouldSendAddressButtons(response string) bool {
	return strings.Contains(response, "Confirma entrega em:")
}

func (p *Processor) sendAddressResponse(ctx context.Context, to, response string) {
	sender, ok := p.whatsApp.(*WhatsAppClient)
	if !ok {
		_ = p.whatsApp.SendMessage(to, response)
		return
	}
	if err := sender.sendButtonsWithIDs(ctx, to, response,
		[]string{"endereco_sim", "endereco_novo"},
		[]string{"Sim", "Novo endereço"}); err != nil {
		logger.Error(ctx, "Erro ao enviar botões de endereço", zap.Error(err))
		_ = p.whatsApp.SendMessage(to, response)
	}
}

func (p *Processor) sendPaymentResponse(ctx context.Context, to, response string) {
	sender, ok := p.whatsApp.(*WhatsAppClient)
	if !ok {
		_ = p.whatsApp.SendMessage(to, response)
		return
	}

	var ids, labels []string
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "*") {
			continue
		}
		end := strings.Index(line[1:], "*")
		if end < 0 {
			continue
		}
		end++
		index, err := strconv.Atoi(line[1:end])
		if err != nil {
			continue
		}
		label := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line[end+1:]), "-"))
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		ids = append(ids, fmt.Sprintf("pagamento_%d", index))
		labels = append(labels, label)
	}
	if len(ids) == 0 || len(ids) > 3 {
		_ = p.whatsApp.SendMessage(to, response)
		return
	}
	if err := sender.sendButtonsWithIDs(ctx, to, response, ids, labels); err != nil {
		logger.Error(ctx, "Erro ao enviar botões de pagamento", zap.Error(err))
		_ = p.whatsApp.SendMessage(to, response)
	}
}

func (p *Processor) sendCartResponse(ctx context.Context, clienteID, tenantID uint, to, response string) {
	sender, ok := p.whatsApp.(buttonSender)
	if !ok || p.cacheLayer == nil {
		_ = p.whatsApp.SendMessage(to, response)
		return
	}
	key := fmt.Sprintf("carrinho:debounce:%d:%d", tenantID, clienteID)
	if err := p.cacheLayer.SetWithContext(ctx, key, response, 5*time.Second); err != nil {
		logger.Error(ctx, "Erro ao armazenar debounce do carrinho", zap.Error(err))
		return
	}
	lockKey := key + ":lock"
	scheduled, err := p.cacheLayer.SetNXWithContext(ctx, lockKey, "1", 5*time.Second)
	if err != nil || !scheduled {
		return
	}
	go func() {
		time.Sleep(2 * time.Second)
		latest, err := p.cacheLayer.GetWithContext(context.Background(), key)
		if err == nil {
			_ = sender.SendButtons(context.Background(), to, latest, []string{"Adicionar mais", "Finalizar pedido", "Limpar"})
		}
		_ = p.cacheLayer.DeleteWithContext(context.Background(), key)
		_ = p.cacheLayer.DeleteWithContext(context.Background(), lockKey)
	}()
}

func (p *Processor) isHumanHandoffActive(ctx context.Context, clienteID uint) bool {
	if p.cacheLayer == nil {
		return false
	}
	_, err := p.cacheLayer.GetWithContext(ctx, fmt.Sprintf("atendimento:humano:%d", clienteID))
	return err == nil
}

func (p *Processor) shouldPromptHandoffTimeout(ctx context.Context, clienteID uint) bool {
	if p.cacheLayer == nil {
		return false
	}
	key := fmt.Sprintf("atendimento:humano:%d", clienteID)
	raw, err := p.cacheLayer.GetWithContext(ctx, key)
	if err != nil {
		return false
	}
	startedAt, err := time.Parse(time.RFC3339, raw)
	if err != nil || time.Since(startedAt) < 10*time.Minute {
		return false
	}
	alertKey := fmt.Sprintf("atendimento:humano:alerta:%d", clienteID)
	set, err := p.cacheLayer.SetNXWithContext(ctx, alertKey, "1", 10*time.Minute)
	return err == nil && set
}

func (p *Processor) forwardToHuman(ctx context.Context, clienteID, tenantID uint, telefone string, input dto.MessageInput) {
	logger.Info(ctx, "mensagem encaminhada para atendimento humano",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("tenant_id", tenantID),
		zap.String("telefone", telefone),
		zap.String("source", string(input.Source)),
	)
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

// NOVO: resolveTenant com phone_number_id (definitivo) + fallback display
func (p *Processor) resolveTenant(ctx context.Context, phoneNumberID, displayPhone string) (uint, error) {
	// 1. Tentativa principal: phone_number_id (vem da Meta, nunca muda)
	if phoneNumberID != "" {
		tenant, err := p.tenantService.GetByWhatsAppPhoneID(ctx, phoneNumberID)
		if err == nil && tenant != nil && tenant.ID != 0 {
			logger.Info(ctx, "tenant resolvido por phone_number_id",
				zap.String("phone_id", phoneNumberID),
				zap.Uint("tenant_id", tenant.ID))
			return tenant.ID, nil
		}
		// se não achou por phoneID, cai pro fallback
		logger.Warn(ctx, "tenant não achado por phone_number_id, tentando fallback display",
			zap.String("phone_id", phoneNumberID), zap.Error(err))
	}

	// 2. Fallback: display phone (seu código antigo)
	clean := phone.Normalize(displayPhone)
	if clean == "" {
		return 0, fmt.Errorf("displayPhone vazio e phoneID %s não resolveu", phoneNumberID)
	}
	tenant, err := p.tenantService.GetByTelefone(ctx, clean)
	if err != nil {
		return 0, err
	}
	if tenant == nil || tenant.ID == 0 {
		return 0, fmt.Errorf("tenant não encontrado para display %s (phoneID %s)", clean, phoneNumberID)
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
		if msg.Type == "interactive" && msg.Interactive.ButtonReply.ID != "" {
			return dto.MessageInput{Source: models.SourceText, Text: msg.Interactive.ButtonReply.ID}
		}
		return dto.MessageInput{Source: models.SourceText, Text: msg.Text.Body}
	}
}
