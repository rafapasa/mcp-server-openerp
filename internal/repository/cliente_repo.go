package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/etoolstec/gokit/apperror"
	"github.com/etoolstec/gokit/filter"
	"github.com/etoolstec/gokit/mapper"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
)

// ClienteRepositoryInterface define as operações do repositório de clientes
type ClienteRepositoryInterface interface {
	GetByID(ctx context.Context, id uint) (*models.Cliente, error)
	GetByTelefone(ctx context.Context, tenantId uint, telefone string) (*models.Cliente, error)
	GetByInscricaoFederal(ctx context.Context, tenantId uint, inscricaoFederal string) (*models.Cliente, error)
	GetByEmail(ctx context.Context, tenantId uint, email string) (*models.Cliente, error)
	FindWithFilters(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]models.Cliente, int64, error)
	Create(ctx context.Context, req dto.CriarClienteRequest) (*models.Cliente, error)
	Update(ctx context.Context, id uint, req dto.AtualizarClienteRequest) (*models.Cliente, error)
	Delete(ctx context.Context, id uint) error
	WithTx(tx *gorm.DB) ClienteRepositoryInterface
}

// clienteRepository implementa o repositório de clientes
type clienteRepository struct {
	db *gorm.DB
}

// NewClienteRepository cria um novo repositório de clientes
func NewClienteRepository(db *gorm.DB) ClienteRepositoryInterface {
	return &clienteRepository{db: db}
}

// WithTx retorna uma nova instância com a transação
func (r *clienteRepository) WithTx(tx *gorm.DB) ClienteRepositoryInterface {
	return &clienteRepository{db: tx}
}

// GetByID busca um cliente pelo ID
func (r *clienteRepository) GetByID(ctx context.Context, id uint) (*models.Cliente, error) {
	var cliente models.Cliente
	err := r.db.WithContext(ctx).
		Preload("Enderecos").
		Preload("Pedidos", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(10)
		}).
		First(&cliente, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("cliente não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar cliente", err)
	}
	return &cliente, nil
}

// GetByTelefone busca um cliente pelo telefone
func (r *clienteRepository) GetByTelefone(ctx context.Context, tenantId uint, telefone string) (*models.Cliente, error) {
	var cliente models.Cliente
	err := r.db.WithContext(ctx).
		Where("enant_id = ? and telefone = ?", tenantId, telefone).
		Preload("Enderecos", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		}).
		First(&cliente).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("cliente não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar cliente por telefone", err)
	}
	return &cliente, nil
}

// GetByInscricaoFederal busca um cliente pela inscrição federal
func (r *clienteRepository) GetByInscricaoFederal(ctx context.Context, tenantId uint, inscricaoFederal string) (*models.Cliente, error) {
	var cliente models.Cliente
	err := r.db.WithContext(ctx).
		Where("enant_id = ? and inscricao_federal = ?", tenantId, inscricaoFederal).
		First(&cliente).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("cliente não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar cliente por inscrição federal", err)
	}
	return &cliente, nil
}

// GetByEmail busca um cliente pelo email
func (r *clienteRepository) GetByEmail(ctx context.Context, tenantId uint, email string) (*models.Cliente, error) {
	var cliente models.Cliente
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? and email = ?", tenantId, email).
		First(&cliente).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("cliente não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar cliente por email", err)
	}
	return &cliente, nil
}

// FindWithFilters busca clientes com filtros e paginação
func (r *clienteRepository) FindWithFilters(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]models.Cliente, int64, error) {
	var clientes []models.Cliente
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Cliente{})
	query = filter.ApplyFilters(query, models.Cliente{}, filters)

	if err := query.Count(&total).Error; err != nil {
		return []models.Cliente{}, 0, apperror.NewInternalError("falha ao contar clientes filtrados", err)
	}

	err := query.
		Preload("Enderecos", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		}).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&clientes).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Cliente{}, total, nil
		}
		return []models.Cliente{}, 0, apperror.NewInternalError("falha ao buscar clientes filtrados", err)
	}
	if clientes == nil {
		clientes = []models.Cliente{}
	}
	return clientes, total, nil
}

// Create cria um novo cliente a partir do DTO
func (r *clienteRepository) Create(ctx context.Context, req dto.CriarClienteRequest) (*models.Cliente, error) {
	cliente := &models.Cliente{}
	if err := mapper.MapToModel(req, cliente); err != nil {
		return nil, apperror.NewInternalError("falha ao converter cliente", err)
	}
	if err := r.db.WithContext(ctx).Create(cliente).Error; err != nil {
		return nil, apperror.NewInternalError("falha ao criar cliente", err)
	}
	return cliente, nil
}

// Update atualiza um cliente a partir do DTO
func (r *clienteRepository) Update(ctx context.Context, id uint, req dto.AtualizarClienteRequest) (*models.Cliente, error) {
	cliente, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := mapper.MapToModel(req, cliente); err != nil {
		return nil, apperror.NewInternalError("falha ao converter cliente", err)
	}
	if err := r.db.WithContext(ctx).Save(cliente).Error; err != nil {
		return nil, apperror.NewInternalError("falha ao atualizar cliente", err)
	}
	return cliente, nil
}

// Delete exclui logicamente um cliente (soft delete)
func (r *clienteRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Cliente{}, id).Error; err != nil {
		return apperror.NewInternalError("falha ao deletar cliente", err)
	}
	return nil
}
