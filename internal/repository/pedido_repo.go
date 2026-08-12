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
	FindByCliente(ctx context.Context, clienteID uint, limit, offset int) ([]models.Pedido, int64, error)
	Create(ctx context.Context, pedido *models.Pedido) error
	Update(ctx context.Context, pedido *models.Pedido) error
	UpdateStatus(ctx context.Context, id uint, status string) (*models.Pedido, error)
	CountByTenantStatus(ctx context.Context, tenantID uint, status string) (int64, error)
	CountByPeriodo(ctx context.Context, tenantID uint, inicio, fim time.Time) (int64, error)
	CountGroupByStatus(ctx context.Context, tenantID uint) (map[string]int64, error)
	CountByStatus(ctx context.Context, tenantID uint, status string) (int64, error)
	SumTotalByPeriodo(ctx context.Context, tenantID uint, inicio, fim time.Time) (float64, error)
	FindWithFilters(ctx context.Context, tenantID uint, clienteID uint, status string, dataInicio, dataFim time.Time, limit, offset int) ([]models.Pedido, int64, error)
}

type pedidoRepository struct {
	db *gorm.DB
}

// NewPedidoRepository creates a new pedido repository instance.
func NewPedidoRepository(db *gorm.DB) PedidoRepository {
	return &pedidoRepository{db: db}
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

func (r *pedidoRepository) Create(ctx context.Context, pedido *models.Pedido) error {
	return r.db.WithContext(ctx).Create(pedido).Error
}

func (r *pedidoRepository) Update(ctx context.Context, pedido *models.Pedido) error {
	return r.db.WithContext(ctx).Save(pedido).Error
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

// internal/repository/pedido_repo.go
// Adicione estes métodos ao PedidoRepository

// ============================================
// DASHBOARD - MÉTRICAS
// ============================================

// CountByPeriodo conta pedidos em um período
func (r *pedidoRepository) CountByPeriodo(ctx context.Context, tenantID uint, inicio, fim time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Pedido{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, inicio, fim).
		Count(&count).Error
	return count, err
}

// CountByStatus conta pedidos por status
func (r *pedidoRepository) CountByStatus(ctx context.Context, tenantID uint, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Pedido{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error
	return count, err
}

// CountGroupByStatus conta pedidos agrupados por status
func (r *pedidoRepository) CountGroupByStatus(ctx context.Context, tenantID uint) (map[string]int64, error) {
	type Result struct {
		Status string
		Count  int64
	}

	var results []Result
	err := r.db.WithContext(ctx).
		Model(&models.Pedido{}).
		Select("status, COUNT(*) as count").
		Where("tenant_id = ?", tenantID).
		Group("status").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	resultMap := make(map[string]int64)
	for _, r := range results {
		resultMap[r.Status] = r.Count
	}
	return resultMap, nil
}

// SumTotalByPeriodo soma o total de pedidos em um período
func (r *pedidoRepository) SumTotalByPeriodo(ctx context.Context, tenantID uint, inicio, fim time.Time) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).
		Model(&models.Pedido{}).
		Select("COALESCE(SUM(total), 0)").
		Where("tenant_id = ? AND created_at BETWEEN ? AND ? AND status IN (?)",
			tenantID, inicio, fim, []string{"confirmado", "entregue"}).
		Scan(&total).Error
	return total, err
}

// ============================================
// FILTERS
// ============================================

// FindWithFilters busca pedidos com filtros e paginação
func (r *pedidoRepository) FindWithFilters(ctx context.Context, tenantID uint, clienteID uint, status string, dataInicio, dataFim time.Time, limit, offset int) ([]models.Pedido, int64, error) {
	var pedidos []models.Pedido
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Pedido{}).Where("tenant_id = ?", tenantID)

	if clienteID > 0 {
		query = query.Where("cliente_id = ?", clienteID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if !dataInicio.IsZero() {
		query = query.Where("created_at >= ?", dataInicio)
	}
	if !dataFim.IsZero() {
		query = query.Where("created_at <= ?", dataFim)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("EnderecoEntrega").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&pedidos).Error

	return pedidos, total, err
}

// FindByCliente busca pedidos de um cliente
func (r *pedidoRepository) FindByCliente(ctx context.Context, clienteID uint, limit, offset int) ([]models.Pedido, int64, error) {
	var pedidos []models.Pedido
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Pedido{}).Where("cliente_id = ?", clienteID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("EnderecoEntrega").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&pedidos).Error

	return pedidos, total, err
}

// FindByID busca um pedido por ID com relacionamentos
func (r *pedidoRepository) FindByID(ctx context.Context, id uint) (*models.Pedido, error) {
	var pedido models.Pedido
	err := r.db.WithContext(ctx).
		Preload("EnderecoEntrega").
		First(&pedido, id).Error
	if err != nil {
		return nil, err
	}
	return &pedido, nil
}

// UpdateStatus atualiza o status de um pedido
func (r *pedidoRepository) UpdateStatus(ctx context.Context, id uint, status string) (*models.Pedido, error) {
	err := r.db.WithContext(ctx).
		Model(&models.Pedido{}).
		Where("id = ?", id).
		Update("status", status).
		Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

// internal/repository/pedido_repo.go
// Adicione estes métodos ao PedidoRepository

// ============================================
// DASHBOARD - MÉTRICAS
// ============================================
