package service

import (
	"context"
	"fmt"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
)

type tenantService struct {
	repo repository.TenantRepository
}

func NewTenantService(repo repository.TenantRepository) TenantServiceInterface {
	return &tenantService{repo: repo}
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
	m := &models.Tenant{Nome: input.Nome, CNPJ: input.CNPJ, Telefone: input.Telefone, Segmento: input.Segmento, Ativo: true}
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
		ID: m.ID, Nome: m.Nome, CNPJ: m.CNPJ,
		Telefone: m.Telefone, Endereco: m.Endereco,
		Segmento: m.Segmento, Ativo: m.Ativo,
	}
}

