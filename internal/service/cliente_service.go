// internal/service/cliente_service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hbollon/go-edlib"
	"github.com/paemuri/brdoc"
	"gorm.io/gorm"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/security"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"go.uber.org/zap"
)

// ClienteService implementa o serviço de clientes
type ClienteService struct {
	clienteRepo  repository.ClienteRepositoryInterface
	enderecoRepo repository.EnderecoRepositoryInterface
}

// NewClienteService cria um novo serviço de clientes
func NewClienteService(
	clienteRepo repository.ClienteRepositoryInterface,
	enderecoRepo repository.EnderecoRepositoryInterface,
) ClienteServiceInterface {
	return &ClienteService{
		clienteRepo:  clienteRepo,
		enderecoRepo: enderecoRepo,
	}
}

// ============================================
// CRUD BÁSICO
// ============================================

// Create cria um novo cliente
func (s *ClienteService) Create(ctx context.Context, req *dto.CriarClienteRequest) (*dto.ClienteDTO, error) {
	// 1. Valida dados
	if err := s.validateCreateRequest(req); err != nil {
		logger.Error(ctx, err.Error())
		return nil, err
	}

	// 2. Verifica se cliente já existe
	clienteDto, err := s.FindByTelefone(ctx, req.Telefone, req.TenantID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("erro ao buscar cliente por telefone: %w", err)
	}

	if clienteDto != nil && clienteDto.ID > 0 {
		// Cliente inativo: reativa e retorna a versão atualizada
		if clienteDto.Status == models.StatusClienteInativo {
			if err := s.ReativarCliente(ctx, clienteDto.ID); err != nil {
				return nil, fmt.Errorf("erro ao reativar cliente: %w", err)
			}
			// Busca de novo para retornar com o status atualizado
			clienteAtualizado, err := s.FindByTelefone(ctx, req.Telefone, req.TenantID)
			if err != nil {
				return nil, err
			}
			return clienteAtualizado, nil
		}
		// Cliente ativo: retorna o existente
		return clienteDto, nil
	}

	// 3. Cria cliente
	cliente := &models.Cliente{
		TenantID:   req.TenantID,
		Telefone:   req.Telefone,
		Nome:       req.Nome,
		NomePerfil: req.Nome,
		Status:     models.StatusClienteAtivo,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 4. Salva no banco
	if err := s.clienteRepo.Create(ctx, cliente); err != nil {
		return nil, fmt.Errorf("erro ao criar cliente: %w", err)
	}

	logger.Info(
		ctx, "Cliente criado com sucesso",
		zap.Uint("cliente_id", cliente.ID),
		zap.String("telefone", cliente.Telefone),
	)

	return s.ConverterParaDTO(cliente), nil
}

// FindByID busca um cliente pelo ID
func (s *ClienteService) FindByID(ctx context.Context, id uint) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("cliente não encontrado")
		}
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	return s.ConverterParaDTO(cliente), nil
}

// FindByTelefone busca um cliente pelo telefone
func (s *ClienteService) FindByTelefone(ctx context.Context, telefone string, tenantID uint) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.FindByTelefone(ctx, telefone, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	return s.ConverterParaDTO(cliente), nil
}

// FindByTenant busca todos os clientes de um tenant
func (s *ClienteService) FindByTenant(ctx context.Context, tenantID uint) ([]dto.ClienteDTO, error) {
	clientes, err := s.clienteRepo.FindByTenant(ctx, fmt.Sprintf("%d", tenantID))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}

	result := make([]dto.ClienteDTO, len(clientes))
	for i, c := range clientes {
		result[i] = *s.ConverterParaDTO(&c)
	}

	return result, nil
}

// Update atualiza um cliente
func (s *ClienteService) Update(ctx context.Context, id uint, req *dto.AtualizarClienteRequest) (*dto.ClienteDTO, error) {
	// 1. Busca cliente
	cliente, err := s.clienteRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("cliente não encontrado: %w", err)
	}

	// 2. Atualiza campos
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
		cliente.GetEnderecoPrincipal().Observacoes = req.Observacoes
	}

	// 3. Salva
	if err := s.clienteRepo.Update(ctx, cliente); err != nil {
		return nil, fmt.Errorf("erro ao atualizar cliente: %w", err)
	}

	logger.Info(
		ctx, "Cliente atualizado",
		zap.Uint("cliente_id", cliente.ID),
	)

	return s.ConverterParaDTO(cliente), nil
}

// Delete exclui logicamente um cliente (soft delete)
func (s *ClienteService) Delete(ctx context.Context, id uint) error {
	// Verifica se cliente existe
	_, err := s.clienteRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("cliente não encontrado: %w", err)
	}

	if err := s.clienteRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("erro ao deletar cliente: %w", err)
	}

	logger.Info(ctx, "Cliente deletado", zap.Uint("cliente_id", id))
	return nil
}

// ============================================
// BUSCAS ESPECÍFICAS
// ============================================

// BuscarOuCriarPorTelefone busca um cliente pelo telefone ou cria um novo
func (s *ClienteService) BuscarOuCriarPorTelefone(ctx context.Context, tenantID uint, telefone, nomePerfil string) (*dto.ClienteDTO, error) {
	// 1. Busca cliente existente
	cliente, err := s.clienteRepo.FindByTelefone(ctx, telefone, tenantID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	// 2. Se não existe, cria novo
	if cliente == nil || cliente.ID == 0 {
		req := &dto.CriarClienteRequest{
			TenantID:   tenantID,
			Telefone:   telefone,
			NomePerfil: nomePerfil,
			Nome:       nomePerfil,
		}
		return s.Create(ctx, req)
	}

	// 3. Se existe, valida o nome
	if cliente.IsAtivo() {
		// Compara o nome perfil com o nome salvo
		if !s.compararNomes(cliente.NomePerfil, nomePerfil) {
			// Se os nomes são diferentes, pergunta ao usuário
			// Aqui você pode implementar a validação interativa
			logger.Warn(
				ctx, "Nome do perfil diferente do salvo",
				zap.String("salvo", cliente.NomePerfil),
				zap.String("atual", nomePerfil),
			)
			// Atualiza o nome perfil se for diferente
			if cliente.NomePerfil != nomePerfil {
				cliente.NomePerfil = nomePerfil
				if err := s.clienteRepo.Update(ctx, cliente); err != nil {
					logger.Warn(ctx, "Erro ao atualizar nome perfil", zap.Error(err))
				}
			}
		}
	}

	// 4. Se o cliente está pendente de validação ou inativo, tenta reativar
	if !cliente.IsAtivo() {
		if err := s.ReativarCliente(ctx, cliente.ID); err != nil {
			return nil, err
		}
	}

	return s.ConverterParaDTO(cliente), nil
}

// BuscarPorNome busca clientes por nome
func (s *ClienteService) BuscarPorNome(ctx context.Context, tenantID uint, nome string) ([]dto.ClienteDTO, error) {
	clientes, err := s.clienteRepo.FindByNome(ctx, fmt.Sprintf("%d", tenantID), nome)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por nome: %w", err)
	}

	result := make([]dto.ClienteDTO, len(clientes))
	for i, c := range clientes {
		result[i] = *s.ConverterParaDTO(&c)
	}

	return result, nil
}

// BuscarPorStatus busca clientes por status
func (s *ClienteService) BuscarPorStatus(ctx context.Context, tenantID uint, status string) ([]dto.ClienteDTO, error) {
	clientes, err := s.clienteRepo.FindByStatus(ctx, fmt.Sprintf("%d", tenantID), status)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por status: %w", err)
	}

	result := make([]dto.ClienteDTO, len(clientes))
	for i, c := range clientes {
		result[i] = *s.ConverterParaDTO(&c)
	}

	return result, nil
}

// BuscarInativos busca clientes inativos há mais de N dias
func (s *ClienteService) BuscarInativos(ctx context.Context, tenantID uint, diasInatividade int) ([]dto.ClienteDTO, error) {
	dataLimite := time.Now().AddDate(0, 0, -diasInatividade)
	clientes, err := s.clienteRepo.FindByUltimoPedidoAntes(ctx, fmt.Sprintf("%d", tenantID), dataLimite)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes inativos: %w", err)
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

// ValidarCliente valida um cliente
func (s *ClienteService) ValidarCliente(ctx context.Context, clienteID uint) (*dto.ClienteDTO, error) {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return nil, fmt.Errorf("cliente não encontrado: %w", err)
	}

	// Verifica se todos os campos obrigatórios estão preenchidos
	if cliente.Nome == "" {
		return nil, fmt.Errorf("nome do cliente não preenchido")
	}

	if cliente.Telefone == "" {
		return nil, fmt.Errorf("telefone do cliente não preenchido")
	}

	return s.ConverterParaDTO(cliente), nil
}

// AtualizarUltimoPedido atualiza a data do último pedido
func (s *ClienteService) AtualizarUltimoPedido(ctx context.Context, clienteID uint) error {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return fmt.Errorf("cliente não encontrado: %w", err)
	}

	cliente.AtualizarUltimoPedido()
	return s.clienteRepo.Update(ctx, cliente)
}

// AtualizarStatus atualiza o status do cliente
func (s *ClienteService) AtualizarStatus(ctx context.Context, clienteID uint, status, motivo string) error {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return fmt.Errorf("cliente não encontrado: %w", err)
	}

	cliente.AtualizarStatus(status, motivo)
	return s.clienteRepo.Update(ctx, cliente)
}

// InativarCliente inativa um cliente
func (s *ClienteService) InativarCliente(ctx context.Context, clienteID uint, motivo string) error {
	return s.AtualizarStatus(ctx, clienteID, models.StatusClienteInativo, motivo)
}

// ReativarCliente reativa um cliente
func (s *ClienteService) ReativarCliente(ctx context.Context, clienteID uint) error {
	return s.AtualizarStatus(ctx, clienteID, models.StatusClienteAtivo, "Reativado após validação")
}

// ============================================
// ENDEREÇOS
// ============================================

// AdicionarEndereco adiciona um endereço ao cliente
func (s *ClienteService) AdicionarEndereco(ctx context.Context, clienteID uint, req *dto.CriarEnderecoRequest) (*dto.EnderecoDTO, error) {
	// 1. Verifica se cliente existe
	_, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return nil, fmt.Errorf("cliente não encontrado: %w", err)
	}

	// 2. Valida endereço
	if req.Logradouro == "" || req.Numero == "" {
		return nil, fmt.Errorf("logradouro e número são obrigatórios")
	}

	// 3. Se for principal, desmarca o principal atual do cliente
	if req.Principal {
		if err := s.enderecoRepo.UnsetPrincipalByCliente(ctx, clienteID); err != nil {
			return nil, fmt.Errorf("erro ao desmarcar endereços principais: %w", err)
		}
	}

	// 4. Cria endereço
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
		return nil, fmt.Errorf("erro ao criar endereço: %w", err)
	}

	logger.Info(
		ctx, "Endereço adicionado ao cliente",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("endereco_id", endereco.ID),
	)

	return s.converterEnderecoDTO(endereco), nil
}

// ListarEnderecos lista os endereços de um cliente
func (s *ClienteService) ListarEnderecos(ctx context.Context, clienteID uint) ([]dto.EnderecoDTO, error) {
	enderecos, err := s.enderecoRepo.FindByClienteAtivos(ctx, clienteID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar endereços: %w", err)
	}

	result := make([]dto.EnderecoDTO, len(enderecos))
	for i, e := range enderecos {
		result[i] = *s.converterEnderecoDTO(&e)
	}

	return result, nil
}

// DefinirEnderecoPrincipal define um endereço como principal
func (s *ClienteService) DefinirEnderecoPrincipal(ctx context.Context, clienteID, enderecoID uint) error {
	// 1. Verifica se endereço existe e pertence ao cliente
	endereco, err := s.enderecoRepo.FindByID(ctx, enderecoID)
	if err != nil {
		return fmt.Errorf("endereço não encontrado: %w", err)
	}
	if endereco.ClienteID != clienteID {
		return fmt.Errorf("endereço não pertence ao cliente")
	}

	// 2. Remove principal de outros endereços do mesmo cliente
	if err := s.enderecoRepo.UnsetPrincipalByCliente(ctx, clienteID); err != nil {
		return fmt.Errorf("erro ao desmarcar endereços principais: %w", err)
	}

	// 3. Define como principal
	endereco.Principal = true
	if err := s.enderecoRepo.Update(ctx, endereco); err != nil {
		return fmt.Errorf("erro ao definir endereço principal: %w", err)
	}

	return nil
}

// RemoverEndereco remove um endereço (soft delete)
func (s *ClienteService) RemoverEndereco(ctx context.Context, clienteID, enderecoID uint) error {
	// Verifica se endereço existe e pertence ao cliente
	endereco, err := s.enderecoRepo.FindByID(ctx, enderecoID)
	if err != nil {
		return fmt.Errorf("endereço não encontrado: %w", err)
	}
	if endereco.ClienteID != clienteID {
		return fmt.Errorf("endereço não pertence ao cliente")
	}

	if err := s.enderecoRepo.Delete(ctx, enderecoID); err != nil {
		return fmt.Errorf("erro ao remover endereço: %w", err)
	}

	return nil
}

// ============================================
// DOCUMENTOS
// ============================================

// AtualizarDocumento atualiza o documento do cliente
func (s *ClienteService) AtualizarDocumento(ctx context.Context, clienteID uint, inscricaoFederal string) error {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return fmt.Errorf("cliente não encontrado: %w", err)
	}

	tipo, err := s.ValidarDocumento(inscricaoFederal)
	if err != nil {
		return err
	}

	cliente.InscricaoFederal = inscricaoFederal
	if err := s.clienteRepo.Update(ctx, cliente); err != nil {
		return fmt.Errorf("erro ao atualizar documento: %w", err)
	}

	logger.Info(
		ctx, "Documento do cliente atualizado",
		zap.Uint("cliente_id", clienteID),
		zap.String("tipo", tipo),
	)

	return nil
}

// ValidarDocumento valida um documento (CPF ou CNPJ)
func (s *ClienteService) ValidarDocumento(inscricaoFederal string) (string, error) {
	// Remove caracteres não numéricos
	doc := strings.ReplaceAll(inscricaoFederal, ".", "")
	doc = strings.ReplaceAll(doc, "-", "")
	doc = strings.ReplaceAll(doc, "/", "")

	if len(doc) == 11 {
		if brdoc.IsCPF(doc) {
			return "fisica", nil
		}
		return "", fmt.Errorf("CPF inválido: %s", inscricaoFederal)
	}

	if len(doc) == 14 {
		if brdoc.IsCNPJ(doc) {
			return "juridica", nil
		}
		return "", fmt.Errorf("CNPJ inválido: %s", inscricaoFederal)
	}

	return "", fmt.Errorf("documento deve ter 11 dígitos (CPF) ou 14 dígitos (CNPJ)")
}

// ============================================
// STATUS
// ============================================

// IsAtivo verifica se um cliente está ativo
func (s *ClienteService) IsAtivo(ctx context.Context, clienteID uint) (bool, error) {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return false, fmt.Errorf("cliente não encontrado: %w", err)
	}
	return cliente.IsAtivo(), nil
}

// GetStatus retorna o status do cliente
func (s *ClienteService) GetStatus(ctx context.Context, clienteID uint) (string, error) {
	cliente, err := s.clienteRepo.FindByID(ctx, clienteID)
	if err != nil {
		return "", fmt.Errorf("cliente não encontrado: %w", err)
	}
	return cliente.Status, nil
}

// ============================================
// UTILITÁRIOS
// ============================================

// ConverterParaDTO converte um model Cliente para DTO
func (s *ClienteService) ConverterParaDTO(cliente *models.Cliente) *dto.ClienteDTO {
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

// converterEnderecoDTO converte um model Endereco para DTO
func (s *ClienteService) converterEnderecoDTO(endereco *models.Endereco) *dto.EnderecoDTO {
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

// ============================================
// MÉTODOS PRIVADOS
// ============================================

// validateCreateRequest valida a requisição de criação
func (s *ClienteService) validateCreateRequest(req *dto.CriarClienteRequest) error {
	if req.TenantID == 0 {
		return fmt.Errorf("tenant_id é obrigatório")
	}
	if req.Telefone == "" {
		return fmt.Errorf("telefone é obrigatório")
	}
	if err := security.ValidatePhoneNumberID(req.Telefone); err != nil {
		return fmt.Errorf("telefone inválido: %w", err)
	}
	if req.Nome == "" {
		req.Nome = req.NomePerfil
	}
	return nil
}

// compararNomes compara dois nomes usando similaridade
func (s *ClienteService) compararNomes(nome1, nome2 string) bool {
	if nome1 == "" || nome2 == "" {
		return false
	}

	// Normaliza os nomes
	nome1 = strings.ToLower(strings.TrimSpace(nome1))
	nome2 = strings.ToLower(strings.TrimSpace(nome2))

	// Se são exatamente iguais
	if nome1 == nome2 {
		return true
	}

	// Usa Jaro-Winkler para similaridade
	similarity := edlib.JaroWinklerSimilarity(nome1, nome2)
	threshold := 0.80 // 80% de similaridade

	return float64(similarity) >= threshold
}

// internal/service/cliente_service.go
// Adicione estes métodos à struct ClienteService

// ============================================
// DASHBOARD METHODS
// ============================================

// CountByTenant conta clientes de um tenant
func (s *ClienteService) CountByTenant(ctx context.Context, tenantID uint) (int64, error) {
	return s.clienteRepo.CountByTenant(ctx, fmt.Sprintf("%d", tenantID))
}

// ============================================
// LIST METHODS
// ============================================

// ListWithFilters lista clientes com filtros e paginação
func (s *ClienteService) ListWithFilters(ctx context.Context, tenantID uint, nome, telefone string, page, limit int) ([]dto.ClienteDTO, int64, error) {
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
