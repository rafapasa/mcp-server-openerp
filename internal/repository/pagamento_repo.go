package repository

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"gorm.io/gorm"
)

type FormaPagamentoRepository interface {
	FindByID(ctx context.Context, id uint) (*models.FormaPagamento, error)
	FindByTenant(ctx context.Context, tenantID uint, apenasAtivas bool) ([]models.FormaPagamento, error)
	Create(ctx context.Context, forma *models.FormaPagamento) error
	Update(ctx context.Context, forma *models.FormaPagamento) error
	Delete(ctx context.Context, id, tenantID uint) error
}

type PedidoPagamentoRepository interface {
	FindByPedido(ctx context.Context, pedidoID uint) ([]models.PedidoPagamento, error)
	CreateMany(ctx context.Context, pagamentos []models.PedidoPagamento) error
	MarcarPendentesComoPagos(ctx context.Context, pedidoID uint) error
}

type formaPagamentoRepository struct{ db *gorm.DB }
type pedidoPagamentoRepository struct{ db *gorm.DB }

func NewFormaPagamentoRepository(db *gorm.DB) FormaPagamentoRepository {
	return &formaPagamentoRepository{db: db}
}

func NewPedidoPagamentoRepository(db *gorm.DB) PedidoPagamentoRepository {
	return &pedidoPagamentoRepository{db: db}
}

func (r *formaPagamentoRepository) FindByID(ctx context.Context, id uint) (*models.FormaPagamento, error) {
	var forma models.FormaPagamento
	if err := r.db.WithContext(ctx).First(&forma, id).Error; err != nil {
		return nil, err
	}
	return &forma, nil
}

func (r *formaPagamentoRepository) FindByTenant(ctx context.Context, tenantID uint, apenasAtivas bool) ([]models.FormaPagamento, error) {
	var formas []models.FormaPagamento
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("nome ASC")
	if apenasAtivas {
		query = query.Where("ativo = ?", true)
	}
	if err := query.Find(&formas).Error; err != nil {
		return nil, err
	}
	return formas, nil
}

func (r *formaPagamentoRepository) Create(ctx context.Context, forma *models.FormaPagamento) error {
	return r.db.WithContext(ctx).Create(forma).Error
}

func (r *formaPagamentoRepository) Update(ctx context.Context, forma *models.FormaPagamento) error {
	return r.db.WithContext(ctx).Model(&models.FormaPagamento{}).
		Where("id = ? AND tenant_id = ?", forma.ID, forma.TenantID).
		Updates(forma).Error
}

func (r *formaPagamentoRepository) Delete(ctx context.Context, id, tenantID uint) error {
	return r.db.WithContext(ctx).Model(&models.FormaPagamento{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Update("ativo", false).Error
}

func (r *pedidoPagamentoRepository) FindByPedido(ctx context.Context, pedidoID uint) ([]models.PedidoPagamento, error) {
	var pagamentos []models.PedidoPagamento
	if err := r.db.WithContext(ctx).
		Preload("FormaPagamento").
		Where("pedido_id = ?", pedidoID).
		Order("id ASC").
		Find(&pagamentos).Error; err != nil {
		return nil, err
	}
	return pagamentos, nil
}

func (r *pedidoPagamentoRepository) CreateMany(ctx context.Context, pagamentos []models.PedidoPagamento) error {
	if len(pagamentos) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&pagamentos).Error
}

func (r *pedidoPagamentoRepository) MarcarPendentesComoPagos(ctx context.Context, pedidoID uint) error {
	return r.db.WithContext(ctx).Model(&models.PedidoPagamento{}).
		Where("pedido_id = ? AND status = ?", pedidoID, models.StatusPagamentoPendente).
		Update("status", models.StatusPagamentoPago).Error
}
