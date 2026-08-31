package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
)

// Adicione este método na sua interface existente (service/interfaces.go)
// type TenantServiceInterface interface {
//   GetByID(ctx context.Context, id uint) (*dto.TenantDTO, error)
//   GetByCNPJ(ctx context.Context, cnpj string) (*dto.TenantDTO, error)
//   GetByTelefone(ctx context.Context, telefone string) (*dto.TenantDTO, error)
//   GetByWhatsAppPhoneID(ctx context.Context, phoneID string) (*dto.TenantDTO, error) // NOVO
//   List(ctx context.Context) ([]dto.TenantDTO, error)
//   Create(ctx context.Context, input dto.CreateTenantDTO) (*dto.TenantDTO, error)
//   GetPromptContext(ctx context.Context, tenantID uint) (string, string, error)
// }

type tenantService struct {
	repo  repository.TenantRepository
	cache database.RedisInterface
}

func NewTenantService(repo repository.TenantRepository, cache database.RedisInterface) TenantServiceInterface {
	return &tenantService{repo: repo, cache: cache}
}

func (s *tenantService) GetByID(ctx context.Context, id uint) (*dto.TenantDTO, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toTenantDTO(m), nil
}

func (s *tenantService) GetByCNPJ(ctx context.Context, cnpj string) (*dto.TenantDTO, error) {
	m, err := s.repo.FindByCNPJ(ctx, cnpj)
	if err != nil {
		return nil, err
	}
	return toTenantDTO(m), nil
}

func (s *tenantService) GetByTelefone(ctx context.Context, telefone string) (*dto.TenantDTO, error) {
	m, err := s.repo.FindByTelefone(ctx, telefone)
	if err != nil {
		return nil, err
	}
	return toTenantDTO(m), nil
}

// NOVO: Método principal para multi-tenant - usa phone_number_id da Meta
func (s *tenantService) GetByWhatsAppPhoneID(ctx context.Context, phoneID string) (*dto.TenantDTO, error) {
	if phoneID == "" {
		return nil, fmt.Errorf("phoneID vazio")
	}
	cacheKey := fmt.Sprintf("tenant:phone:%s", phoneID)

	// Usa seu GetOrSet genérico já existente em database/redis.go
	tenantDTO, err := database.GetOrSet(s.cache, ctx, cacheKey, 1*time.Hour, func() (*dto.TenantDTO, error) {
		m, err := s.repo.FindByWhatsAppPhoneID(ctx, phoneID)
		if err != nil {
			return nil, err
		}
		return toTenantDTO(m), nil
	})
	if err != nil {
		return nil, err
	}
	return tenantDTO, nil
}

func (s *tenantService) List(ctx context.Context) ([]dto.TenantDTO, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.TenantDTO, len(list))
	for i, t := range list {
		out[i] = *toTenantDTO(&t)
	}
	return out, nil
}

func (s *tenantService) Create(ctx context.Context, input dto.CreateTenantDTO) (*dto.TenantDTO, error) {
	m := &models.Tenant{
		Nome:                  input.Nome,
		CNPJ:                  input.CNPJ,
		Telefone:              input.Telefone,
		Segmento:              input.Segmento,
		WabaID:                input.WabaID,
		WhatsappPhoneID:       input.WhatsappPhoneID,
		WhatsappDisplayNumber: input.WhatsappDisplayNumber,
		Ativo:                 true,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return toTenantDTO(m), nil
}

// Método que tava perdido no webhook_handler.go
func (s *tenantService) GetPromptContext(ctx context.Context, tenantID uint) (string, string, error) {
	m, err := s.repo.FindByID(ctx, tenantID)
	if err != nil {
		return "", "", fmt.Errorf("tenant %d não encontrado", tenantID)
	}
	seg := m.Segmento
	if seg == "" {
		seg = "geral"
	}
	return m.Nome, seg, nil
}

func toTenantDTO(m *models.Tenant) *dto.TenantDTO {
	return &dto.TenantDTO{
		ID:                    m.ID,
		Nome:                  m.Nome,
		CNPJ:                  m.CNPJ,
		Telefone:              m.Telefone,
		Endereco:              m.Endereco,
		Segmento:              m.Segmento,
		WabaID:                m.WabaID,
		WhatsappPhoneID:       m.WhatsappPhoneID,
		WhatsappDisplayNumber: m.WhatsappDisplayNumber,
		Ativo:                 m.Ativo,
	}
}
