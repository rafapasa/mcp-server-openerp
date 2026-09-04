package repository

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

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

	// Contagem
	CountByTenant(ctx context.Context, tenantID string) (int64, error)
	CountByStatus(ctx context.Context, tenantID string, status string) (int64, error)

	// Transações
	WithTx(tx *gorm.DB) ClienteRepositoryInterface
	FindWithFilters(ctx context.Context, tenantID uint, nome, telefone string, limit, offset int) ([]models.Cliente, int64, error)
}

// ClienteRepository implementa o repositório de clientes
type ClienteRepository struct {
	db *gorm.DB
}

// NewClienteRepository cria um novo repositório de clientes
func NewClienteRepository(db *gorm.DB) ClienteRepositoryInterface {
	return &ClienteRepository{db: db}
}

// WithTx retorna uma nova instância com a transação
func (r *ClienteRepository) WithTx(tx *gorm.DB) ClienteRepositoryInterface {
	return &ClienteRepository{db: tx}
}

// ============================================
// CRUD BÁSICO
// ============================================

// Create cria um novo cliente
func (r *ClienteRepository) Create(ctx context.Context, cliente *models.Cliente) error {
	return r.db.WithContext(ctx).Create(cliente).Error
}

// FindByID busca um cliente pelo ID
func (r *ClienteRepository) FindByID(ctx context.Context, id uint) (*models.Cliente, error) {
	var cliente models.Cliente
	err := r.db.WithContext(ctx).
		Preload("Enderecos").
		Preload("Pedidos", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(10)
		}).
		First(&cliente, id).Error
	if err != nil {
		return nil, err
	}
	return &cliente, nil
}

// FindByTelefone busca um cliente pelo telefone
func (r *ClienteRepository) FindByTelefone(ctx context.Context, telefone string, tenantID uint) (*models.Cliente, error) {
	var cliente models.Cliente
	err := r.db.WithContext(ctx).
		Where("telefone = ? AND tenant_id = ?", telefone, tenantID).
		Preload("Enderecos", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		}).
		First(&cliente).Error
	if err != nil {
		logger.Error(
			ctx, "erro executando consulta SQL",
			// zap.String("SQL", r.db.Commit().Statement.TableExpr.SQL),
			zap.String("Erro", err.Error()),
		)
		return nil, err
	}
	return &cliente, nil
}

// FindByTenant busca todos os clientes de um tenant
func (r *ClienteRepository) FindByTenant(ctx context.Context, tenantID string) ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Preload("Enderecos", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		}).
		Order("created_at DESC").
		Find(&clientes).Error
	return clientes, err
}

// Update atualiza um cliente
func (r *ClienteRepository) Update(ctx context.Context, cliente *models.Cliente) error {
	return r.db.WithContext(ctx).Save(cliente).Error
}

// Delete exclui logicamente um cliente (soft delete)
// Mantém histórico, apenas marca como inativo
func (r *ClienteRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Cliente{}, id).Error
}

// ============================================
// BUSCAS ESPECÍFICAS
// ============================================

// FindByStatus busca clientes por status
func (r *ClienteRepository) FindByStatus(ctx context.Context, tenantID string, status string) ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Order("ultimo_pedido_at DESC").
		Find(&clientes).Error
	return clientes, err
}

// FindByNome busca clientes por nome (case insensitive)
func (r *ClienteRepository) FindByNome(ctx context.Context, tenantID string, nome string) ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND nome LIKE ?", tenantID, "%"+nome+"%").
		Order("nome ASC").
		Find(&clientes).Error
	return clientes, err
}

// FindByUltimoPedidoAntes busca clientes que não fizeram pedidos desde a data
func (r *ClienteRepository) FindByUltimoPedidoAntes(ctx context.Context, tenantID string, data time.Time) ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND (ultimo_pedido_at IS NULL OR ultimo_pedido_at < ?)", tenantID, data).
		Find(&clientes).Error
	return clientes, err
}

// ============================================
// CONTAGEM
// ============================================

// CountByTenant conta clientes de um tenant
func (r *ClienteRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Cliente{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error
	return count, err
}

// CountByStatus conta clientes por status
func (r *ClienteRepository) CountByStatus(ctx context.Context, tenantID string, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Cliente{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error
	return count, err
}

// internal/repository/cliente_repo.go
// Adicione estes métodos ao ClienteRepository

// FindWithFilters busca clientes com filtros e paginação
func (r *ClienteRepository) FindWithFilters(ctx context.Context, tenantID uint, nome, telefone string, limit, offset int) ([]models.Cliente, int64, error) {
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
		return nil, 0, err
	}

	err := query.
		Preload("Enderecos", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		}).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&clientes).Error

	return clientes, total, err
}
