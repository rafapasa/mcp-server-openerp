package repository

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/etoolstec/gokit/apperror"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
)

// ClienteRepositoryInterface define as operações do repositório de clientes
type ClienteRepositoryInterface interface {
	// CRUD Básico
	Create(ctx context.Context, cliente *models.Cliente) error
	FindByID(ctx context.Context, id uint) (*models.Cliente, error)
	FindByTelefone(ctx context.Context, telefone string, tenantID uint) (*models.Cliente, error)
	FindByTenant(ctx context.Context, tenantID string) ([]models.Cliente, error)
	Update(ctx context.Context, cliente *models.Cliente) error
	Delete(ctx context.Context, id uint) error

	// Buscas Específicas
	FindByStatus(ctx context.Context, tenantID string, status string) ([]models.Cliente, error)
	FindByNome(ctx context.Context, tenantID string, nome string) ([]models.Cliente, error)
	FindByUltimoPedidoAntes(ctx context.Context, tenantID string, data time.Time) ([]models.Cliente, error)
	FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]models.Cliente, int64, error)

	// Contagem
	CountByTenant(ctx context.Context, tenantID string) (int64, error)
	CountByStatus(ctx context.Context, tenantID string, status string) (int64, error)

	// Transações
	WithTx(tx *gorm.DB) ClienteRepositoryInterface
	FindWithFilters(ctx context.Context, tenantID uint, nome, telefone string, limit, offset int) ([]models.Cliente, int64, error)
}

// ClienteRepository implementa o repositório de clientes
type clienteRepository struct {
	db *gorm.DB
}

// NewClienteRepository cria um novo repositório de clientes
func NewClienteRepository(db *gorm.DB) ClienteRepositoryInterface {
	return &clienteRepository{db: db}
}

func (r *clienteRepository) FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]models.Cliente, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	var clientes []models.Cliente
	var total int64

	err := r.db.WithContext(ctx).
		Model(&models.Cliente{}).
		Where("tenant_id = ?", tenantID).
		Count(&total).Error
	if err != nil {
		return nil, 0, apperror.NewInternalError("falha ao contar clientes", err)
	}

	err = r.db.WithContext(ctx).
		Model(&models.Cliente{}).
		Where("tenant_id = ?", tenantID).
		Offset(offset).
		Limit(limit).
		Find(&clientes).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Cliente{}, total, nil
		}
		return nil, 0, apperror.NewInternalError("falha ao buscar clientes", err)
	}
	if clientes == nil {
		clientes = []models.Cliente{}
	}
	return clientes, total, nil
}

// WithTx retorna uma nova instância com a transação
func (r *clienteRepository) WithTx(tx *gorm.DB) ClienteRepositoryInterface {
	return &clienteRepository{db: tx}
}

// ============================================
// CRUD BÁSICO
// ============================================

// Create cria um novo cliente
func (r *clienteRepository) Create(ctx context.Context, cliente *models.Cliente) error {
	if err := r.db.WithContext(ctx).Create(cliente).Error; err != nil {
		return apperror.NewInternalError("falha ao criar cliente", err)
	}
	return nil
}

// FindByID busca um cliente pelo ID
func (r *clienteRepository) FindByID(ctx context.Context, id uint) (*models.Cliente, error) {
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

// FindByTelefone busca um cliente pelo telefone
func (r *clienteRepository) FindByTelefone(ctx context.Context, telefone string, tenantID uint) (*models.Cliente, error) {
	var cliente models.Cliente
	err := r.db.WithContext(ctx).
		Where("telefone = ? AND tenant_id = ?", telefone, tenantID).
		Preload("Enderecos", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		}).
		First(&cliente).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("cliente não encontrado")
		}
		logger.Error(
			ctx, "erro executando consulta SQL",
			zap.String("Erro", err.Error()),
		)
		return nil, apperror.NewInternalError("falha ao buscar cliente por telefone", err)
	}
	return &cliente, nil
}

// FindByTenant busca todos os clientes de um tenant
func (r *clienteRepository) FindByTenant(ctx context.Context, tenantID string) ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Preload("Enderecos", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		}).
		Order("created_at DESC").
		Find(&clientes).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Cliente{}, nil
		}
		return nil, apperror.NewInternalError("falha ao listar clientes", err)
	}
	return clientes, nil
}

// Update atualiza um cliente
func (r *clienteRepository) Update(ctx context.Context, cliente *models.Cliente) error {
	if err := r.db.WithContext(ctx).Save(cliente).Error; err != nil {
		return apperror.NewInternalError("falha ao atualizar cliente", err)
	}
	return nil
}

// Delete exclui logicamente um cliente (soft delete)
// Mantém histórico, apenas marca como inativo
func (r *clienteRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Cliente{}, id).Error; err != nil {
		return apperror.NewInternalError("falha ao deletar cliente", err)
	}
	return nil
}

// ============================================
// BUSCAS ESPECÍFICAS
// ============================================

// FindByStatus busca clientes por status
func (r *clienteRepository) FindByStatus(ctx context.Context, tenantID string, status string) ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Order("ultimo_pedido_at DESC").
		Find(&clientes).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Cliente{}, nil
		}
		return nil, apperror.NewInternalError("falha ao buscar clientes por status", err)
	}
	return clientes, nil
}

// FindByNome busca clientes por nome (case insensitive)
func (r *clienteRepository) FindByNome(ctx context.Context, tenantID string, nome string) ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND nome LIKE ?", tenantID, "%"+nome+"%").
		Order("nome ASC").
		Find(&clientes).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Cliente{}, nil
		}
		return nil, apperror.NewInternalError("falha ao buscar clientes por nome", err)
	}
	return clientes, nil
}

// FindByUltimoPedidoAntes busca clientes que não fizeram pedidos desde a data
func (r *clienteRepository) FindByUltimoPedidoAntes(ctx context.Context, tenantID string, data time.Time) ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND (ultimo_pedido_at IS NULL OR ultimo_pedido_at < ?)", tenantID, data).
		Find(&clientes).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Cliente{}, nil
		}
		return nil, apperror.NewInternalError("falha ao buscar clientes por último pedido", err)
	}
	return clientes, nil
}

// ============================================
// CONTAGEM
// ============================================

// CountByTenant conta clientes de um tenant
func (r *clienteRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Cliente{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error
	if err != nil {
		return 0, apperror.NewInternalError("falha ao contar clientes", err)
	}
	return count, nil
}

// CountByStatus conta clientes por status
func (r *clienteRepository) CountByStatus(ctx context.Context, tenantID string, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Cliente{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error
	if err != nil {
		return 0, apperror.NewInternalError("falha ao contar clientes por status", err)
	}
	return count, nil
}

// FindWithFilters busca clientes com filtros e paginação
func (r *clienteRepository) FindWithFilters(ctx context.Context, tenantID uint, nome, telefone string, limit, offset int) ([]models.Cliente, int64, error) {
	var clientes []models.Cliente
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Cliente{}).Where("tenant_id = ?", tenantID)

	if nome != "" {
		query = query.Where("nome LIKE ?", "%"+nome+"%")
	}
	if telefone != "" {
		query = query.Where("telefone LIKE ?", "%"+telefone+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.NewInternalError("falha ao contar clientes filtrados", err)
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
		return nil, 0, apperror.NewInternalError("falha ao buscar clientes filtrados", err)
	}
	if clientes == nil {
		clientes = []models.Cliente{}
	}
	return clientes, total, nil
}
