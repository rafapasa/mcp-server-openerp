package repository

import (
	"context"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"gorm.io/gorm"
)

// PedidoRepository defines the contract for accessing and persisting pedidos.
type PedidoRepository interface {
	FindByID(ctx context.Context, id uint) (*models.Pedido, error)
	FindByTenant(ctx context.Context, tenantID uint, limit, offset int) ([]models.Pedido, int64, error)
	FindByTenantStatus(ctx context.Context, tenantID uint, status string) ([]models.Pedido, error)
	FindByTenantPeriodo(ctx context.Context, tenantID uint, inicio, fim time.Time) ([]models.Pedido, error)
	FindByCliente(ctx context.Context, tenantID uint, clienteID string) ([]models.Pedido, error)
	Create(ctx context.Context, pedido *models.Pedido) error
	Update(ctx context.Context, pedido *models.Pedido) error
	UpdateStatus(ctx context.Context, id uint, status string) error
	CountByTenantStatus(ctx context.Context, tenantID uint, status string) (int64, error)
}

type pedidoRepository struct {
	db *gorm.DB
}

// NewPedidoRepository creates a new pedido repository instance.
func NewPedidoRepository(db *gorm.DB) PedidoRepository {
	return &pedidoRepository{db: db}
}

func (r *pedidoRepository) FindByID(ctx context.Context, id uint) (*models.Pedido, error) {
	var pedido models.Pedido
	if err := r.db.WithContext(ctx).
		Preload("Tenant").
		First(&pedido, id).Error; err != nil {
		return nil, err
	}
	return &pedido, nil
}

func (r *pedidoRepository) FindByTenant(ctx context.Context, tenantID uint, limit, offset int) ([]models.Pedido, int64, error) {
	var pedidos []models.Pedido
	var total int64

	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	// Conta total
	if err := query.Model(&models.Pedido{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Busca com paginação
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&pedidos).Error; err != nil {
		return nil, 0, err
	}

	return pedidos, total, nil
}

func (r *pedidoRepository) FindByTenantStatus(ctx context.Context, tenantID uint, status string) ([]models.Pedido, error) {
	var pedidos []models.Pedido
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Order("created_at DESC").
		Find(&pedidos).Error; err != nil {
		return nil, err
	}
	return pedidos, nil
}

func (r *pedidoRepository) FindByTenantPeriodo(ctx context.Context, tenantID uint, inicio, fim time.Time) ([]models.Pedido, error) {
	var pedidos []models.Pedido
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, inicio, fim).
		Order("created_at DESC").
		Find(&pedidos).Error; err != nil {
		return nil, err
	}
	return pedidos, nil
}

func (r *pedidoRepository) FindByCliente(ctx context.Context, tenantID uint, clienteID string) ([]models.Pedido, error) {
	var pedidos []models.Pedido
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND cliente_id = ?", tenantID, clienteID).
		Order("created_at DESC").
		Find(&pedidos).Error; err != nil {
		return nil, err
	}
	return pedidos, nil
}

func (r *pedidoRepository) Create(ctx context.Context, pedido *models.Pedido) error {
	return r.db.WithContext(ctx).Create(pedido).Error
}

func (r *pedidoRepository) Update(ctx context.Context, pedido *models.Pedido) error {
	return r.db.WithContext(ctx).Save(pedido).Error
}

func (r *pedidoRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.Pedido{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *pedidoRepository) CountByTenantStatus(ctx context.Context, tenantID uint, status string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Pedido{}).Where("tenant_id = ?", tenantID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
