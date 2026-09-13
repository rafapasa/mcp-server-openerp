package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/etoolstec/gokit/apperror"
	"github.com/etoolstec/gokit/filter"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
)

type TenantRepository interface {
	GetByID(ctx context.Context, id uint) (*models.Tenant, error)
	GetByCNPJ(ctx context.Context, cnpj string) (*models.Tenant, error)
	GetByTelefone(ctx context.Context, telefone string) (*models.Tenant, error)
	GetByWhatsAppPhoneID(ctx context.Context, phoneID string) (*models.Tenant, error)
	GetByVerifyToken(ctx context.Context, token string) (*models.Tenant, error)
	FindWithFilters(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]models.Tenant, int64, error)
	Create(ctx context.Context, tenant *models.Tenant) error
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, id uint) error
}

type tenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) GetByID(ctx context.Context, id uint) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).First(&tenant, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) GetByCNPJ(ctx context.Context, cnpj string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("cnpj = ?", cnpj).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant por CNPJ", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) GetByTelefone(ctx context.Context, telefone string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("telefone = ?", telefone).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant por telefone", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) GetByWhatsAppPhoneID(ctx context.Context, phoneID string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("whatsapp_phone_id = ?", phoneID).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant por WhatsApp phone ID", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) GetByVerifyToken(ctx context.Context, token string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("whatsapp_verify_token = ?", token).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant por verify token", err)
	}
	return &tenant, nil
}

// GetByTenantPaginated lista os tenants de forma paginada.
// Como o próprio tenant é a raiz do isolamento multi-tenant, o filtro por
// tenantID é opcional: quando informado (> 0), restringe ao tenant de mesmo ID.
func (r *tenantRepository) FindWithFilters(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]models.Tenant, int64, error) {
	var tenants []models.Tenant
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Tenant{})
	query = filter.ApplyFilters(query, models.Tenant{}, filters)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.NewInternalError("falha ao contar tenants", err)
	}

	err := query.
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&tenants).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Tenant{}, total, nil
		}
		return nil, 0, apperror.NewInternalError("falha ao listar tenants", err)
	}
	if tenants == nil {
		tenants = []models.Tenant{}
	}
	return tenants, total, err
}

func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	if err := r.db.WithContext(ctx).Create(tenant).Error; err != nil {
		return apperror.NewInternalError("falha ao criar tenant", err)
	}
	return nil
}

func (r *tenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	if err := r.db.WithContext(ctx).Save(tenant).Error; err != nil {
		return apperror.NewInternalError("falha ao atualizar tenant", err)
	}
	return nil
}

func (r *tenantRepository) Delete(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&models.Tenant{}, id)
	if res.Error != nil {
		return apperror.NewInternalError("falha ao deletar tenant", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.NewNotFoundError("tenant não encontrado")
	}
	return nil
}
