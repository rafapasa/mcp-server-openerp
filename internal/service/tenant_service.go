package service

import (
	"context"
	"fmt"
	"time"

	"github.com/etoolstec/gokit/apperror"
	"github.com/etoolstec/gokit/mapper"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
)

type tenantService struct {
	repo  repository.TenantRepository
	cache database.RedisInterface
}

func NewTenantService(repo repository.TenantRepository, cache database.RedisInterface) TenantServiceInterface {
	return &tenantService{repo: repo, cache: cache}
}

func (s *tenantService) GetByID(ctx context.Context, id uint) (*dto.TenantDTO, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toTenantDTO(m), nil
}

func (s *tenantService) GetByCNPJ(ctx context.Context, cnpj string) (*dto.TenantDTO, error) {
	m, err := s.repo.GetByCNPJ(ctx, cnpj)
	if err != nil {
		return nil, err
	}
	return toTenantDTO(m), nil
}

func (s *tenantService) GetByTelefone(ctx context.Context, telefone string) (*dto.TenantDTO, error) {
	m, err := s.repo.GetByTelefone(ctx, telefone)
	if err != nil {
		return nil, err
	}
	return toTenantDTO(m), nil
}

func (s *tenantService) GetByWhatsAppPhoneID(ctx context.Context, phoneID string) (*dto.TenantDTO, error) {
	if phoneID == "" {
		return nil, apperror.NewBadRequestError("phoneID vazio")
	}
	cacheKey := fmt.Sprintf("tenant:phone:%s", phoneID)
	tenantDTO, err := database.GetOrSet(s.cache, ctx, cacheKey, 1*time.Hour, func() (*dto.TenantDTO, error) {
		m, err := s.repo.GetByWhatsAppPhoneID(ctx, phoneID)
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

func (s *tenantService) GetByVerifyToken(ctx context.Context, token string) (*dto.TenantDTO, error) {
	if token == "" {
		return nil, apperror.NewBadRequestError("verify token vazio")
	}
	cacheKey := fmt.Sprintf("tenant:verify:%s", token)
	tenantDTO, err := database.GetOrSet(s.cache, ctx, cacheKey, 1*time.Hour, func() (*dto.TenantDTO, error) {
		m, err := s.repo.GetByVerifyToken(ctx, token)
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

func (s *tenantService) List(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]dto.TenantDTO, int64, error) {
	tenants, total, err := s.repo.FindWithFilters(ctx, limit, offset, filters)
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.TenantDTO, len(tenants))
	for i, t := range tenants {
		if err := mapper.MapToDTO(t, &result[i]); err != nil {
			logger.Error(ctx, fmt.Sprintf("Erro convertendo Model Tenant para DTO tenant: %v", err))
			return nil, 0, apperror.NewInternalError("Erro convertendo Model Tenant para DTO tenant", err)
		}
	}
	return result, total, nil
}

func (s *tenantService) Create(ctx context.Context, input dto.CreateTenantDTO) (*dto.TenantDTO, error) {
	m := &models.Tenant{
		Nome:                  input.Nome,
		CNPJ:                  input.CNPJ,
		Telefone:              input.Telefone,
		Endereco:              input.Endereco,
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

func (s *tenantService) Update(ctx context.Context, id uint, input dto.UpdateTenantDTO) (*dto.TenantDTO, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Nome != nil {
		existing.Nome = *input.Nome
	}
	if input.CNPJ != nil {
		existing.CNPJ = *input.CNPJ
	}
	if input.Telefone != nil {
		existing.Telefone = *input.Telefone
	}
	if input.Endereco != nil {
		existing.Endereco = *input.Endereco
	}
	if input.Segmento != nil {
		existing.Segmento = *input.Segmento
	}
	if input.WabaID != nil {
		existing.WabaID = *input.WabaID
	}
	if input.WhatsappPhoneID != nil {
		existing.WhatsappPhoneID = *input.WhatsappPhoneID
	}
	if input.WhatsappDisplayNumber != nil {
		existing.WhatsappDisplayNumber = *input.WhatsappDisplayNumber
	}
	if input.Ativo != nil {
		existing.Ativo = *input.Ativo
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.DeleteWithContext(ctx, fmt.Sprintf("tenant:phone:%s", existing.WhatsappPhoneID))
		_ = s.cache.DeleteWithContext(ctx, fmt.Sprintf("tenant:verify:%s", existing.WhatsappVerifyToken))
	}

	return toTenantDTO(existing), nil
}

func (s *tenantService) Delete(ctx context.Context, id uint) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.cache != nil {
		_ = s.cache.DeleteWithContext(ctx, fmt.Sprintf("tenant:phone:%s", existing.WhatsappPhoneID))
		_ = s.cache.DeleteWithContext(ctx, fmt.Sprintf("tenant:verify:%s", existing.WhatsappVerifyToken))
	}
	return nil
}

func (s *tenantService) GetPromptContext(ctx context.Context, tenantID uint) (string, string, error) {
	m, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return "", "", err
	}
	seg := m.Segmento
	if seg == "" {
		seg = "geral"
	}
	return m.Nome, seg, nil
}

func (s *tenantService) FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]dto.TenantDTO, int64, error) {
	filters := map[string]any{"tenant_id": tenantID}
	offset := (page - 1) * limit
	return s.List(ctx, limit, offset, filters)
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
