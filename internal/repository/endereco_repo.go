package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/etoolstec/gokit/apperror"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
)

// EnderecoRepositoryInterface define as operações do repositório de endereços
type EnderecoRepositoryInterface interface {
	// CRUD Básico
	Create(ctx context.Context, endereco *models.Endereco) error
	FindByID(ctx context.Context, id uint) (*models.Endereco, error)
	FindByCliente(ctx context.Context, clienteID uint) ([]models.Endereco, error)
	FindByClienteAtivos(ctx context.Context, clienteID uint) ([]models.Endereco, error)
	Update(ctx context.Context, endereco *models.Endereco) error
	Delete(ctx context.Context, id uint) error
	DeletePermanente(ctx context.Context, id uint) error

	// Buscas Específicas
	FindPrincipal(ctx context.Context, clienteID uint) (*models.Endereco, error)
	FindByCEP(ctx context.Context, cep string) ([]models.Endereco, error)
	FindByClienteETipo(ctx context.Context, clienteID uint, tipo string) ([]models.Endereco, error)

	// Contagem
	CountByCliente(ctx context.Context, clienteID uint) (int64, error)
	CountAtivosByCliente(ctx context.Context, clienteID uint) (int64, error)

	// Transações
	WithTx(tx *gorm.DB) EnderecoRepositoryInterface

	// Gestão de principal
	UnsetPrincipalByCliente(ctx context.Context, clienteID uint) error
}

// EnderecoRepository implementa o repositório de endereços
type enderecoRepository struct {
	db *gorm.DB
}

// NewEnderecoRepository cria um novo repositório de endereços
func NewEnderecoRepository(db *gorm.DB) EnderecoRepositoryInterface {
	return &enderecoRepository{db: db}
}

// WithTx retorna uma nova instância com a transação
func (r *enderecoRepository) WithTx(tx *gorm.DB) EnderecoRepositoryInterface {
	return &enderecoRepository{db: tx}
}

// ============================================
// CRUD BÁSICO
// ============================================

// Create cria um novo endereço
func (r *enderecoRepository) Create(ctx context.Context, endereco *models.Endereco) error {
	if err := r.db.WithContext(ctx).Create(endereco).Error; err != nil {
		return apperror.NewInternalError("falha ao criar endereço", err)
	}
	return nil
}

// FindByID busca um endereço pelo ID
func (r *enderecoRepository) FindByID(ctx context.Context, id uint) (*models.Endereco, error) {
	var endereco models.Endereco
	err := r.db.WithContext(ctx).
		Preload("Cliente").
		First(&endereco, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("endereço não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar endereço", err)
	}
	return &endereco, nil
}

// FindByCliente busca todos os endereços de um cliente
func (r *enderecoRepository) FindByCliente(ctx context.Context, clienteID uint) ([]models.Endereco, error) {
	var enderecos []models.Endereco
	err := r.db.WithContext(ctx).
		Where("cliente_id = ?", clienteID).
		Order("principal DESC, created_at DESC").
		Find(&enderecos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Endereco{}, nil
		}
		return nil, apperror.NewInternalError("falha ao listar endereços do cliente", err)
	}
	return enderecos, nil
}

// FindByClienteAtivos busca apenas os endereços ativos de um cliente
func (r *enderecoRepository) FindByClienteAtivos(ctx context.Context, clienteID uint) ([]models.Endereco, error) {
	var enderecos []models.Endereco
	err := r.db.WithContext(ctx).
		Where("cliente_id = ? AND deleted_at IS NULL", clienteID).
		Order("principal DESC, created_at DESC").
		Find(&enderecos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Endereco{}, nil
		}
		return nil, apperror.NewInternalError("falha ao listar endereços ativos", err)
	}
	return enderecos, nil
}

// Update atualiza um endereço
// NOTA: Endereços são imutáveis! Use Delete + Create para "editar"
func (r *enderecoRepository) Update(ctx context.Context, endereco *models.Endereco) error {
	if err := r.db.WithContext(ctx).Save(endereco).Error; err != nil {
		return apperror.NewInternalError("falha ao atualizar endereço", err)
	}
	return nil
}

// Delete realiza soft delete do endereço
func (r *enderecoRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Endereco{}, id).Error; err != nil {
		return apperror.NewInternalError("falha ao deletar endereço", err)
	}
	return nil
}

// DeletePermanente remove permanentemente o endereço (cuidado!)
func (r *enderecoRepository) DeletePermanente(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Unscoped().Delete(&models.Endereco{}, id).Error; err != nil {
		return apperror.NewInternalError("falha ao deletar endereço permanentemente", err)
	}
	return nil
}

// ============================================
// BUSCAS ESPECÍFICAS
// ============================================

// FindPrincipal busca o endereço principal de um cliente
func (r *enderecoRepository) FindPrincipal(ctx context.Context, clienteID uint) (*models.Endereco, error) {
	var endereco models.Endereco
	err := r.db.WithContext(ctx).
		Where("cliente_id = ? AND principal = ? AND deleted_at IS NULL", clienteID, true).
		First(&endereco).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("endereço principal não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar endereço principal", err)
	}
	return &endereco, nil
}

// FindByCEP busca endereços por CEP
func (r *enderecoRepository) FindByCEP(ctx context.Context, cep string) ([]models.Endereco, error) {
	var enderecos []models.Endereco
	err := r.db.WithContext(ctx).
		Where("cep = ?", cep).
		Preload("Cliente").
		Find(&enderecos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Endereco{}, nil
		}
		return nil, apperror.NewInternalError("falha ao buscar endereços por CEP", err)
	}
	return enderecos, nil
}

// FindByClienteETipo busca endereços de um cliente por tipo
func (r *enderecoRepository) FindByClienteETipo(ctx context.Context, clienteID uint, tipo string) ([]models.Endereco, error) {
	var enderecos []models.Endereco
	err := r.db.WithContext(ctx).
		Where("cliente_id = ? AND tipo = ? AND deleted_at IS NULL", clienteID, tipo).
		Order("principal DESC, created_at DESC").
		Find(&enderecos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Endereco{}, nil
		}
		return nil, apperror.NewInternalError("falha ao buscar endereços por tipo", err)
	}
	return enderecos, nil
}

// ============================================
// CONTAGEM
// ============================================

// CountByCliente conta o total de endereços de um cliente
func (r *enderecoRepository) CountByCliente(ctx context.Context, clienteID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Endereco{}).
		Where("cliente_id = ?", clienteID).
		Count(&count).Error
	if err != nil {
		return 0, apperror.NewInternalError("falha ao contar endereços", err)
	}
	return count, nil
}

// CountAtivosByCliente conta os endereços ativos de um cliente
func (r *enderecoRepository) CountAtivosByCliente(ctx context.Context, clienteID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Endereco{}).
		Where("cliente_id = ? AND deleted_at IS NULL", clienteID).
		Count(&count).Error
	if err != nil {
		return 0, apperror.NewInternalError("falha ao contar endereços ativos", err)
	}
	return count, nil
}

// ============================================
// GESTÃO DE PRINCIPAL
// ============================================

// UnsetPrincipalByCliente desmarca o flag principal de todos os endereços ativos de um cliente
func (r *enderecoRepository) UnsetPrincipalByCliente(ctx context.Context, clienteID uint) error {
	if err := r.db.WithContext(ctx).
		Model(&models.Endereco{}).
		Where("cliente_id = ? AND deleted_at IS NULL", clienteID).
		Update("principal", false).Error; err != nil {
		return apperror.NewInternalError("falha ao desmarcar endereços principais", err)
	}
	return nil
}
