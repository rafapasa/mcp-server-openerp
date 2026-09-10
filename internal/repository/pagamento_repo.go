package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/etoolstec/gokit/apperror"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("forma de pagamento não encontrada")
		}
		return nil, apperror.NewInternalError("falha ao buscar forma de pagamento", err)
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.FormaPagamento{}, nil
		}
		return nil, apperror.NewInternalError("falha ao listar formas de pagamento", err)
	}
	return formas, nil
}

func (r *formaPagamentoRepository) Create(ctx context.Context, forma *models.FormaPagamento) error {
	if err := r.db.WithContext(ctx).Create(forma).Error; err != nil {
		return apperror.NewInternalError("falha ao criar forma de pagamento", err)
	}
	return nil
}

func (r *formaPagamentoRepository) Update(ctx context.Context, forma *models.FormaPagamento) error {
	if err := r.db.WithContext(ctx).Model(&models.FormaPagamento{}).
		Where("id = ? AND tenant_id = ?", forma.ID, forma.TenantID).
		Updates(forma).Error; err != nil {
		return apperror.NewInternalError("falha ao atualizar forma de pagamento", err)
	}
	return nil
}

func (r *formaPagamentoRepository) Delete(ctx context.Context, id, tenantID uint) error {
	if err := r.db.WithContext(ctx).Model(&models.FormaPagamento{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Update("ativo", false).Error; err != nil {
		return apperror.NewInternalError("falha ao desativar forma de pagamento", err)
	}
	return nil
}

func (r *pedidoPagamentoRepository) FindByPedido(ctx context.Context, pedidoID uint) ([]models.PedidoPagamento, error) {
	var pagamentos []models.PedidoPagamento
	if err := r.db.WithContext(ctx).
		Preload("FormaPagamento").
		Where("pedido_id = ?", pedidoID).
		Order("id ASC").
		Find(&pagamentos).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.PedidoPagamento{}, nil
		}
		return nil, apperror.NewInternalError("falha ao buscar pagamentos do pedido", err)
	}
	return pagamentos, nil
}

func (r *pedidoPagamentoRepository) CreateMany(ctx context.Context, pagamentos []models.PedidoPagamento) error {
	if len(pagamentos) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&pagamentos).Error; err != nil {
		return apperror.NewInternalError("falha ao criar pagamentos", err)
	}
	return nil
}

func (r *pedidoPagamentoRepository) MarcarPendentesComoPagos(ctx context.Context, pedidoID uint) error {
	if err := r.db.WithContext(ctx).Model(&models.PedidoPagamento{}).
		Where("pedido_id = ? AND status = ?", pedidoID, models.StatusPagamentoPendente).
		Update("status", models.StatusPagamentoPago).Error; err != nil {
		return apperror.NewInternalError("falha ao marcar pagamentos como pagos", err)
	}
	return nil
}
