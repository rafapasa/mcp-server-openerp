package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hbollon/go-edlib"

	"github.com/etoolstec/gokit/apperror"
	"github.com/etoolstec/gokit/document"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/security"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"go.uber.org/zap"
)

type clienteService struct {
	clienteRepo  repository.ClienteRepositoryInterface
	enderecoRepo repository.EnderecoRepositoryInterface
}

func NewClienteService(
	clienteRepo repository.ClienteRepositoryInterface,
	enderecoRepo repository.EnderecoRepositoryInterface,
) ClienteServiceInterface {
	return &clienteService{
		clienteRepo:  clienteRepo,
		enderecoRepo: enderecoRepo,
	}
}

func isNotFound(err error) bool {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == 404
	}
	return false
}

// ============================================
// CRUD BÁSICO
// ============================================

func (s *clienteService) Create(ctx context.Context, req *dto.CriarClienteRequest) (*dto.ClienteDTO, error) {
	if err := s.validateCreateRequest(req); err != nil {
		logger.Error(ctx, err.Error())
		return nil, err
	}

	clienteDto, err := s.FindByTelefone(ctx, req.Telefone, req.TenantID)
	if err != nil {
		return nil, err
	}

	if clienteDto != nil && clienteDto.ID > 0 {
		if clienteDto.Status == models.StatusClienteInativo {
			if err := s.ReativarCliente(ctx, clienteDto.ID); err != nil {
				return nil, err
			}
			clienteAtualizado, err := s.FindByTelefone(ctx, req.Telefone, req.TenantID)
			if err != nil {
				return nil, err
			}
			return clienteAtualizado, nil
		}
		return clienteDto, nil
	}

	cliente := &models.Cliente{
		TenantID:   req.TenantID,
		Telefone:   req.Telefone,
		Nome:       req.Nome,
		NomePerfil: req.Nome,
		Status:     models.StatusClienteAtivo,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.clienteRepo.Create(ctx, cliente); err != nil {
		return nil, err
	}

	logger.Info(
		ctx, "Cliente criado com sucesso",
		zap.Uint("cliente_id", cliente.ID),
		zap.String("telefone", cliente.Telefone),
	)

	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) FindByID(ctx context.Context, id uint) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) FindByTelefone(ctx context.Context, telefone string, tenantID uint) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.FindByTelefone(ctx, telefone, tenantID)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) FindByTenant(ctx context.Context, tenantID uint) ([]dto.ClienteDTO, error) {
	clientes, err := s.clienteRepo.FindByTenant(ctx, fmt.Sprintf("%d", tenantID))
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i, c := range clientes {
		result[i] = *s.ConverterParaDTO(&c)
	}
	return result, nil
}

func (s *clienteService) Update(ctx context.Context, id uint, req *dto.AtualizarClienteRequest) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Nome != "" {
		cliente.Nome = req.Nome
	}
	if req.Email != "" {
		cliente.Email = req.Email
	}
	if req.InscricaoFederal != "" {
		if _, err := s.ValidarDocumento(req.InscricaoFederal); err != nil {
			return nil, err
		}
		cliente.InscricaoFederal = req.InscricaoFederal
	}
	if req.Observacoes != "" {
		if ep := cliente.GetEnderecoPrincipal(); ep != nil {
			ep.Observacoes = req.Observacoes
		}
	}

	if err := s.clienteRepo.Update(ctx, cliente); err != nil {
		return nil, err
	}

	logger.Info(ctx, "Cliente atualizado", zap.Uint("cliente_id", cliente.ID))
	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) Delete(ctx context.Context, id uint) error {
	_, err := s.clienteRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.clienteRepo.Delete(ctx, id); err != nil {
		return err
	}
	logger.Info(ctx, "Cliente deletado", zap.Uint("cliente_id", id))
	return nil
}

// ============================================
// BUSCAS ESPECÍFICAS
// ============================================

func (s *clienteService) BuscarOuCriarPorTelefone(ctx context.Context, tenantID uint, telefone, nomePerfil string) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.FindByTelefone(ctx, telefone, tenantID)
	if err != nil {
		if !isNotFound(err) {
			return nil, err
		}
		cliente = nil
	}

	if cliente == nil || cliente.ID == 0 {
		req := &dto.CriarClienteRequest{
			TenantID:   tenantID,
			Telefone:   telefone,
			NomePerfil: nomePerfil,
			Nome:       nomePerfil,
		}
		return s.Create(ctx, req)
	}

	if cliente.IsAtivo() {
		if !s.compararNomes(cliente.NomePerfil, nomePerfil) {
			logger.Warn(ctx, "Nome do perfil diferente do salvo",
				zap.String("salvo", cliente.NomePerfil),
				zap.String("atual", nomePerfil),
			)
			if cliente.NomePerfil != nomePerfil {
				cliente.NomePerfil = nomePerfil
				if err := s.clienteRepo.Update(ctx, cliente); err != nil {
					logger.Warn(ctx, "Erro ao atualizar nome perfil", zap.Error(err))
				}
			}
		}
	}

	if !cliente.IsAtivo() {
		if err := s.ReativarCliente(ctx, cliente.ID); err != nil {
			return nil, err
		}
	}

	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) BuscarPorNome(ctx context.Context, tenantID uint, nome string) ([]dto.ClienteDTO, error) {
	clientes, err := s.clienteRepo.FindByNome(ctx, fmt.Sprintf("%d", tenantID), nome)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i, c := range clientes {
		result[i] = *s.ConverterParaDTO(&c)
	}
	return result, nil
}

func (s *clienteService) BuscarPorStatus(ctx context.Context, tenantID uint, status string) ([]dto.ClienteDTO, error) {
	clientes, err := s.clienteRepo.FindByStatus(ctx, fmt.Sprintf("%d", tenantID), status)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i, c := range clientes {
		result[i] = *s.ConverterParaDTO(&c)
	}
	return result, nil
}

func (s *clienteService) BuscarInativos(ctx context.Context, tenantID uint, diasInatividade int) ([]dto.ClienteDTO, error) {
	dataLimite := time.Now().AddDate(0, 0, -diasInatividade)
	clientes, err := s.clienteRepo.FindByUltimoPedidoAntes(ctx, fmt.Sprintf("%d", tenantID), dataLimite)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i, c := range clientes {
		result[i] = *s.ConverterParaDTO(&c)
	}
	return result, nil
}

// ============================================
// VALIDAÇÃO E GESTÃO
// ============================================

func (s *clienteService) ValidarCliente(ctx context.Context, clienteID uint) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return nil, err
	}
	if cliente.Nome == "" {
		return nil, apperror.NewValidationError("nome do cliente não preenchido")
	}
	if cliente.Telefone == "" {
		return nil, apperror.NewValidationError("telefone do cliente não preenchido")
	}
	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) AtualizarUltimoPedido(ctx context.Context, clienteID uint) error {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return err
	}
	cliente.AtualizarUltimoPedido()
	return s.clienteRepo.Update(ctx, cliente)
}

func (s *clienteService) AtualizarStatus(ctx context.Context, clienteID uint, status, motivo string) error {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return err
	}
	cliente.AtualizarStatus(status, motivo)
	return s.clienteRepo.Update(ctx, cliente)
}

func (s *clienteService) InativarCliente(ctx context.Context, clienteID uint, motivo string) error {
	return s.AtualizarStatus(ctx, clienteID, models.StatusClienteInativo, motivo)
}

func (s *clienteService) ReativarCliente(ctx context.Context, clienteID uint) error {
	return s.AtualizarStatus(ctx, clienteID, models.StatusClienteAtivo, "Reativado após validação")
}

// ============================================
// ENDEREÇOS
// ============================================

func (s *clienteService) AdicionarEndereco(ctx context.Context, clienteID uint, req *dto.CriarEnderecoRequest) (*dto.EnderecoDTO, error) {
	_, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return nil, err
	}

	if req.Logradouro == "" || req.Numero == "" {
		return nil, apperror.NewBadRequestError("logradouro e número são obrigatórios")
	}

	if req.Principal {
		if err := s.enderecoRepo.UnsetPrincipalByCliente(ctx, clienteID); err != nil {
			return nil, err
		}
	}

	endereco := &models.Endereco{
		ClienteID:   clienteID,
		CEP:         req.CEP,
		Logradouro:  req.Logradouro,
		Numero:      req.Numero,
		Complemento: req.Complemento,
		Bairro:      req.Bairro,
		Cidade:      req.Cidade,
		Estado:      req.Estado,
		Pais:        req.Pais,
		Referencia:  req.Referencia,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Tipo:        req.Tipo,
		Principal:   req.Principal,
	}

	if err := s.enderecoRepo.Create(ctx, endereco); err != nil {
		return nil, err
	}

	logger.Info(ctx, "Endereço adicionado ao cliente",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("endereco_id", endereco.ID),
	)

	return s.converterEnderecoDTO(endereco), nil
}

func (s *clienteService) ListarEnderecos(ctx context.Context, clienteID uint) ([]dto.EnderecoDTO, error) {
	enderecos, err := s.enderecoRepo.FindByClienteAtivos(ctx, clienteID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.EnderecoDTO, len(enderecos))
	for i, e := range enderecos {
		result[i] = *s.converterEnderecoDTO(&e)
	}
	return result, nil
}

func (s *clienteService) DefinirEnderecoPrincipal(ctx context.Context, clienteID, enderecoID uint) error {
	endereco, err := s.enderecoRepo.FindByID(ctx, enderecoID)
	if err != nil {
		return err
	}
	if endereco.ClienteID != clienteID {
		return apperror.NewForbiddenError("endereço não pertence ao cliente")
	}

	if err := s.enderecoRepo.UnsetPrincipalByCliente(ctx, clienteID); err != nil {
		return err
	}

	endereco.Principal = true
	if err := s.enderecoRepo.Update(ctx, endereco); err != nil {
		return err
	}
	return nil
}

func (s *clienteService) RemoverEndereco(ctx context.Context, clienteID, enderecoID uint) error {
	endereco, err := s.enderecoRepo.FindByID(ctx, enderecoID)
	if err != nil {
		return err
	}
	if endereco.ClienteID != clienteID {
		return apperror.NewForbiddenError("endereço não pertence ao cliente")
	}
	if err := s.enderecoRepo.Delete(ctx, enderecoID); err != nil {
		return err
	}
	return nil
}

// ============================================
// DOCUMENTOS
// ============================================

func (s *clienteService) AtualizarDocumento(ctx context.Context, clienteID uint, inscricaoFederal string) error {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return err
	}
	tipo, err := s.ValidarDocumento(inscricaoFederal)
	if err != nil {
		return err
	}
	cliente.InscricaoFederal = inscricaoFederal
	if err := s.clienteRepo.Update(ctx, cliente); err != nil {
		return err
	}
	logger.Info(ctx, "Documento do cliente atualizado",
		zap.Uint("cliente_id", clienteID),
		zap.String("tipo", tipo),
	)
	return nil
}

// ValidarDocumento usa gokit/document (limpeza + CPF/CNPJ).
// Retorna "fisica" | "juridica" ou erro de validação.
func (s *clienteService) ValidarDocumento(inscricaoFederal string) (string, error) {
	docV := document.NewDocumentValidator()
	doc := docV.LimparDocumento(inscricaoFederal)

	if len(doc) == 11 {
		if docV.IsValidCPF(doc) {
			return "fisica", nil
		}
		return "", apperror.NewBadRequestError(fmt.Sprintf("CPF inválido: %s", inscricaoFederal))
	}
	if len(doc) == 14 {
		if docV.IsValidCNPJ(doc) {
			return "juridica", nil
		}
		return "", apperror.NewBadRequestError(fmt.Sprintf("CNPJ inválido: %s", inscricaoFederal))
	}
	return "", apperror.NewBadRequestError("documento deve ter 11 dígitos (CPF) ou 14 dígitos (CNPJ)")
}

// ============================================
// STATUS
// ============================================

func (s *clienteService) IsAtivo(ctx context.Context, clienteID uint) (bool, error) {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return false, err
	}
	return cliente.IsAtivo(), nil
}

func (s *clienteService) GetStatus(ctx context.Context, clienteID uint) (string, error) {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return "", err
	}
	return cliente.Status, nil
}

// ============================================
// UTILITÁRIOS
// ============================================

func (s *clienteService) ConverterParaDTO(cliente *models.Cliente) *dto.ClienteDTO {
	if cliente == nil {
		return nil
	}
	var enderecos []dto.EnderecoDTO
	if len(cliente.Enderecos) > 0 {
		enderecos = make([]dto.EnderecoDTO, len(cliente.Enderecos))
		for i, e := range cliente.Enderecos {
			enderecos[i] = *s.converterEnderecoDTO(&e)
		}
	}
	return &dto.ClienteDTO{
		ID:                  cliente.ID,
		TenantID:            cliente.TenantID,
		Telefone:            cliente.Telefone,
		Nome:                cliente.Nome,
		NomePerfil:          cliente.NomePerfil,
		Email:               cliente.Email,
		InscricaoFederal:    cliente.InscricaoFederal,
		RG:                  cliente.RG,
		InscricaoEstadual:   cliente.InscricaoEstadual,
		InscricaoMunicipal:  cliente.InscricaoMunicipal,
		Status:              cliente.Status,
		StatusReason:        cliente.StatusReason,
		StatusUpdatedAt:     cliente.StatusUpdatedAt,
		NomeAnterior:        cliente.NomeAnterior,
		UltimaValidacaoNome: cliente.UltimaValidacaoNome,
		UltimoPedidoAt:      cliente.UltimoPedidoAt,
		Enderecos:           enderecos,
		CreatedAt:           cliente.CreatedAt,
		UpdatedAt:           cliente.UpdatedAt,
	}
}

func (s *clienteService) converterEnderecoDTO(endereco *models.Endereco) *dto.EnderecoDTO {
	if endereco == nil {
		return nil
	}
	return &dto.EnderecoDTO{
		ID:          endereco.ID,
		ClienteID:   endereco.ClienteID,
		CEP:         endereco.CEP,
		Logradouro:  endereco.Logradouro,
		Numero:      endereco.Numero,
		Complemento: endereco.Complemento,
		Bairro:      endereco.Bairro,
		Cidade:      endereco.Cidade,
		Estado:      endereco.Estado,
		Pais:        endereco.Pais,
		Referencia:  endereco.Referencia,
		Latitude:    endereco.Latitude,
		Longitude:   endereco.Longitude,
		Tipo:        endereco.Tipo,
		Principal:   endereco.Principal,
		Observacoes: endereco.Observacoes,
		CreatedAt:   endereco.CreatedAt,
		UpdatedAt:   endereco.UpdatedAt,
	}
}

func (s *clienteService) validateCreateRequest(req *dto.CriarClienteRequest) error {
	if req.TenantID == 0 {
		return apperror.NewBadRequestError("tenant_id é obrigatório")
	}
	if req.Telefone == "" {
		return apperror.NewBadRequestError("telefone é obrigatório")
	}
	if err := security.ValidatePhoneNumberID(req.Telefone); err != nil {
		return apperror.NewBadRequestError(fmt.Sprintf("telefone inválido: %s", err.Error()))
	}
	if req.Nome == "" {
		req.Nome = req.NomePerfil
	}
	return nil
}

func (s *clienteService) compararNomes(nome1, nome2 string) bool {
	if nome1 == "" || nome2 == "" {
		return false
	}
	nome1 = strings.ToLower(strings.TrimSpace(nome1))
	nome2 = strings.ToLower(strings.TrimSpace(nome2))
	if nome1 == nome2 {
		return true
	}
	similarity := edlib.JaroWinklerSimilarity(nome1, nome2)
	threshold := 0.80
	return float64(similarity) >= threshold
}

func (s *clienteService) CountByTenant(ctx context.Context, tenantID uint) (int64, error) {
	return s.clienteRepo.CountByTenant(ctx, fmt.Sprintf("%d", tenantID))
}

func (s *clienteService) ListWithFilters(ctx context.Context, tenantID uint, nome, telefone string, page, limit int) ([]dto.ClienteDTO, int64, error) {
	offset := (page - 1) * limit
	clientes, total, err := s.clienteRepo.FindWithFilters(ctx, tenantID, nome, telefone, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i, c := range clientes {
		result[i] = *s.ConverterParaDTO(&c)
	}
	return result, total, nil
}

func (s *clienteService) FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]dto.ClienteDTO, int64, error) {
	offset := (page - 1) * limit
	clientes, total, err := s.clienteRepo.FindByTenantPaginated(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i, c := range clientes {
		result[i] = *s.ConverterParaDTO(&c)
	}
	return result, total, nil
}
