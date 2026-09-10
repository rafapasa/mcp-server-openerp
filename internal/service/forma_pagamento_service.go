package service

import (
	"context"
	"strings"

	"github.com/etoolstec/gokit/apperror"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
)

var tiposFormaPagamento = map[string]struct{}{
	models.TipoPagamentoDinheiro:      {},
	models.TipoPagamentoPix:           {},
	models.TipoPagamentoCartaoCredito: {},
	models.TipoPagamentoCartaoDebito:  {},
}

type formaPagamentoService struct {
	repo repository.FormaPagamentoRepository
}

func NewFormaPagamentoService(repo repository.FormaPagamentoRepository) FormaPagamentoServiceInterface {
	return &formaPagamentoService{repo: repo}
}

func (s *formaPagamentoService) Listar(ctx context.Context, tenantID uint, apenasAtivas bool) ([]dto.FormaPagamentoDTO, error) {
	formas, err := s.repo.FindByTenant(ctx, tenantID, apenasAtivas)
	if err != nil {
		return nil, err
	}
	result := make([]dto.FormaPagamentoDTO, len(formas))
	for i := range formas {
		result[i] = formaPagamentoDTO(&formas[i])
	}
	return result, nil
}

func (s *formaPagamentoService) Buscar(ctx context.Context, tenantID, id uint) (*dto.FormaPagamentoDTO, error) {
	forma, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if forma.TenantID != tenantID {
		return nil, apperror.NewForbiddenError("forma de pagamento não pertence ao tenant")
	}
	result := formaPagamentoDTO(forma)
	return &result, nil
}

func (s *formaPagamentoService) Criar(ctx context.Context, tenantID uint, req dto.CriarFormaPagamentoRequest) (*dto.FormaPagamentoDTO, error) {
	nome, tipo, err := validarFormaPagamento(req.Nome, req.Tipo)
	if err != nil {
		return nil, err
	}
	forma := &models.FormaPagamento{TenantID: tenantID, Nome: nome, Tipo: tipo, Ativo: true}
	if err := s.repo.Create(ctx, forma); err != nil {
		return nil, err
	}
	result := formaPagamentoDTO(forma)
	return &result, nil
}

func (s *formaPagamentoService) Atualizar(ctx context.Context, tenantID, id uint, req dto.AtualizarFormaPagamentoRequest) (*dto.FormaPagamentoDTO, error) {
	forma, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if forma.TenantID != tenantID {
		return nil, apperror.NewForbiddenError("forma de pagamento não pertence ao tenant")
	}
	nome, tipo, err := validarFormaPagamento(req.Nome, req.Tipo)
	if err != nil {
		return nil, err
	}
	forma.Nome = nome
	forma.Tipo = tipo
	if req.Ativo != nil {
		forma.Ativo = *req.Ativo
	}
	if err := s.repo.Update(ctx, forma); err != nil {
		return nil, err
	}
	result := formaPagamentoDTO(forma)
	return &result, nil
}

func (s *formaPagamentoService) Inativar(ctx context.Context, tenantID, id uint) error {
	forma, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if forma.TenantID != tenantID {
		return apperror.NewForbiddenError("forma de pagamento não pertence ao tenant")
	}
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		return err
	}
	return nil
}

func validarFormaPagamento(nome, tipo string) (string, string, error) {
	nome = strings.TrimSpace(nome)
	tipo = strings.TrimSpace(strings.ToLower(tipo))
	if nome == "" || len([]rune(nome)) > 100 {
		return "", "", apperror.NewBadRequestError("nome da forma de pagamento deve ter entre 1 e 100 caracteres")
	}
	if _, ok := tiposFormaPagamento[tipo]; !ok {
		return "", "", apperror.NewBadRequestError("tipo de forma de pagamento inválido")
	}
	return nome, tipo, nil
}

func formaPagamentoDTO(forma *models.FormaPagamento) dto.FormaPagamentoDTO {
	return dto.FormaPagamentoDTO{
		ID: forma.ID, TenantID: forma.TenantID, Nome: forma.Nome,
		Tipo: forma.Tipo, Ativo: forma.Ativo,
	}
}
