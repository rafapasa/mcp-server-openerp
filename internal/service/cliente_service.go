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
	"github.com/etoolstec/gokit/mapper"
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
// FLUXO HANDLER -> REPO
// ============================================

func (s *clienteService) List(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]dto.ClienteDTO, int64, error) {
	clientes, total, err := s.clienteRepo.FindWithFilters(ctx, limit, offset, filters)
	if err != nil {
		return nil, 0, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i := range clientes {
		result[i] = *s.ConverterParaDTO(&clientes[i])
	}
	return result, total, nil
}

func (s *clienteService) GetByID(ctx context.Context, id uint) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) GetByTelefone(ctx context.Context, tenantId uint, telefone string) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.GetByTelefone(ctx, tenantId, telefone)
	if err != nil {
		return nil, err
	}
	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) GetByInscricaoFederal(ctx context.Context, tenantId uint, inscricaoFederal string) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.GetByInscricaoFederal(ctx, tenantId, inscricaoFederal)
	if err != nil {
		return nil, err
	}
	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) GetByEmail(ctx context.Context, tenantId uint, email string) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.GetByEmail(ctx, tenantId, email)
	if err != nil {
		return nil, err
	}
	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) Create(ctx context.Context, req *dto.CriarClienteRequest) (*dto.ClienteDTO, error) {
	if err := s.validateCreateRequest(req); err != nil {
		logger.Error(ctx, err.Error())
		return nil, err
	}

	clienteExistente, err := s.clienteRepo.GetByTelefone(ctx, req.TenantID, req.Telefone)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	if clienteExistente != nil && clienteExistente.ID > 0 {
		if clienteExistente.Status == models.StatusClienteInativo {
			if err := s.ReativarCliente(ctx, clienteExistente.ID); err != nil {
				return nil, err
			}
			clienteAtualizado, err := s.clienteRepo.GetByTelefone(ctx, req.TenantID, req.Telefone)
			if err != nil {
				return nil, err
			}
			return s.ConverterParaDTO(clienteAtualizado), nil
		}
		return s.ConverterParaDTO(clienteExistente), nil
	}

	cliente, err := s.clienteRepo.Create(ctx, *req)
	if err != nil {
		return nil, err
	}

	logger.Info(
		ctx, "Cliente criado com sucesso",
		zap.Uint("cliente_id", cliente.ID),
		zap.String("telefone", cliente.Telefone),
	)

	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) Update(ctx context.Context, id uint, req *dto.AtualizarClienteRequest) (*dto.ClienteDTO, error) {
	if req.InscricaoFederal != "" {
		if _, err := s.ValidarDocumento(req.InscricaoFederal); err != nil {
			return nil, err
		}
	}

	if req.Observacoes != "" {
		if err := s.atualizarObservacoesEnderecoPrincipal(ctx, id, req.Observacoes); err != nil {
			return nil, err
		}
	}

	cliente, err := s.clienteRepo.Update(ctx, id, *req)
	if err != nil {
		return nil, err
	}

	logger.Info(ctx, "Cliente atualizado", zap.Uint("cliente_id", cliente.ID))
	return s.ConverterParaDTO(cliente), nil
}

// atualizarObservacoesEnderecoPrincipal aplica as observações no endereço
// principal do cliente (se existir) e persiste a alteração via enderecoRepo.
func (s *clienteService) atualizarObservacoesEnderecoPrincipal(ctx context.Context, clienteID uint, observacoes string) error {
	enderecos, err := s.enderecoRepo.FindByClienteAtivos(ctx, clienteID)
	if err != nil {
		return err
	}
	for i := range enderecos {
		if enderecos[i].Principal {
			enderecos[i].Observacoes = observacoes
			return s.enderecoRepo.Update(ctx, &enderecos[i])
		}
	}
	return nil
}

func (s *clienteService) Delete(ctx context.Context, id uint) error {
	if _, err := s.clienteRepo.GetByID(ctx, id); err != nil {
		return err
	}
	if err := s.clienteRepo.Delete(ctx, id); err != nil {
		return err
	}
	logger.Info(ctx, "Cliente deletado", zap.Uint("cliente_id", id))
	return nil
}

func (s *clienteService) FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]dto.ClienteDTO, int64, error) {
	offset := (page - 1) * limit
	filters := map[string]interface{}{"tenant_id": tenantID}
	return s.List(ctx, limit, offset, filters)
}

// ============================================
// BUSCAS ESPECÍFICAS
// ============================================

func (s *clienteService) FindByTenant(ctx context.Context, tenantID uint) ([]dto.ClienteDTO, error) {
	filters := map[string]interface{}{"tenant_id": tenantID}
	clientes, _, err := s.clienteRepo.FindWithFilters(ctx, 0, 0, filters)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i := range clientes {
		result[i] = *s.ConverterParaDTO(&clientes[i])
	}
	return result, nil
}

func (s *clienteService) BuscarOuCriarPorTelefone(ctx context.Context, tenantID uint, telefone, nomePerfil string) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.GetByTelefone(ctx, tenantID, telefone)
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
				req := &dto.AtualizarClienteRequest{NomePerfil: nomePerfil}
				if _, err := s.clienteRepo.Update(ctx, cliente.ID, *req); err != nil {
					logger.Warn(ctx, "Erro ao atualizar nome perfil", zap.Error(err))
				}
			}
		}
	}

	if !cliente.IsAtivo() {
		if err := s.ReativarCliente(ctx, cliente.ID); err != nil {
			return nil, err
		}
		// Recarrega para que a resposta reflita o status reativado.
		cliente, err = s.clienteRepo.GetByTelefone(ctx, tenantID, telefone)
		if err != nil {
			return nil, err
		}
	}

	return s.ConverterParaDTO(cliente), nil
}

func (s *clienteService) BuscarPorNome(ctx context.Context, tenantID uint, nome string) ([]dto.ClienteDTO, error) {
	filters := map[string]interface{}{
		"tenant_id": tenantID,
		"nome":      nome,
	}
	clientes, _, err := s.clienteRepo.FindWithFilters(ctx, 0, 0, filters)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i := range clientes {
		result[i] = *s.ConverterParaDTO(&clientes[i])
	}
	return result, nil
}

func (s *clienteService) BuscarPorStatus(ctx context.Context, tenantID uint, status string) ([]dto.ClienteDTO, error) {
	filters := map[string]interface{}{
		"tenant_id": tenantID,
		"status":    status,
	}
	clientes, _, err := s.clienteRepo.FindWithFilters(ctx, 0, 0, filters)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i := range clientes {
		result[i] = *s.ConverterParaDTO(&clientes[i])
	}
	return result, nil
}

func (s *clienteService) BuscarInativos(ctx context.Context, tenantID uint, diasInatividade int) ([]dto.ClienteDTO, error) {
	dataLimite := time.Now().AddDate(0, 0, -diasInatividade)
	filters := map[string]interface{}{
		"tenant_id":           tenantID,
		"ultimo_pedido_at_lt": dataLimite,
	}
	clientes, _, err := s.clienteRepo.FindWithFilters(ctx, 0, 0, filters)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClienteDTO, len(clientes))
	for i := range clientes {
		result[i] = *s.ConverterParaDTO(&clientes[i])
	}
	return result, nil
}

// ============================================
// VALIDAÇÃO E GESTÃO
// ============================================

func (s *clienteService) ValidarCliente(ctx context.Context, clienteID uint) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.GetByID(ctx, clienteID)
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
	now := time.Now()
	req := &dto.AtualizarClienteRequest{UltimoPedidoAt: &now}
	if _, err := s.clienteRepo.Update(ctx, clienteID, *req); err != nil {
		return err
	}
	return nil
}

func (s *clienteService) AtualizarStatus(ctx context.Context, clienteID uint, status, motivo string) error {
	now := time.Now()
	req := &dto.AtualizarClienteRequest{
		Status:          status,
		StatusReason:    motivo,
		StatusUpdatedAt: &now,
	}
	if _, err := s.clienteRepo.Update(ctx, clienteID, *req); err != nil {
		return err
	}
	return nil
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
	_, err := s.clienteRepo.GetByID(ctx, clienteID)
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

	endereco := &models.Endereco{}
	if err := mapper.MapToModel(req, endereco); err != nil {
		return nil, apperror.NewInternalError("falha ao converter endereço", err)
	}
	endereco.ClienteID = clienteID

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
	for i := range enderecos {
		result[i] = *s.converterEnderecoDTO(&enderecos[i])
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
	tipo, err := s.ValidarDocumento(inscricaoFederal)
	if err != nil {
		return err
	}
	req := &dto.AtualizarClienteRequest{InscricaoFederal: inscricaoFederal}
	if _, err := s.clienteRepo.Update(ctx, clienteID, *req); err != nil {
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
	cliente, err := s.clienteRepo.GetByID(ctx, clienteID)
	if err != nil {
		return false, err
	}
	return cliente.IsAtivo(), nil
}

func (s *clienteService) GetStatus(ctx context.Context, clienteID uint) (string, error) {
	cliente, err := s.clienteRepo.GetByID(ctx, clienteID)
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
	result := &dto.ClienteDTO{}
	if err := mapper.MapToDTO(cliente, result); err != nil {
		return nil
	}
	if len(cliente.Enderecos) > 0 {
		result.Enderecos = make([]dto.EnderecoDTO, len(cliente.Enderecos))
		for i := range cliente.Enderecos {
			result.Enderecos[i] = *s.converterEnderecoDTO(&cliente.Enderecos[i])
		}
	}
	return result
}

func (s *clienteService) converterEnderecoDTO(endereco *models.Endereco) *dto.EnderecoDTO {
	if endereco == nil {
		return nil
	}
	result := &dto.EnderecoDTO{}
	if err := mapper.MapToDTO(endereco, result); err != nil {
		return nil
	}
	return result
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
	if req.NomePerfil == "" {
		req.NomePerfil = req.Nome
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
	filters := map[string]interface{}{"tenant_id": tenantID}
	_, total, err := s.clienteRepo.FindWithFilters(ctx, 0, 0, filters)
	if err != nil {
		return 0, err
	}
	return total, nil
}
