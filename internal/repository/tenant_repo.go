package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/etoolstec/gokit/apperror"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
)

type TenantRepository interface {
	FindByID(ctx context.Context, id uint) (*models.Tenant, error)
	FindByCNPJ(ctx context.Context, cnpj string) (*models.Tenant, error)
	FindByTelefone(ctx context.Context, telefone string) (*models.Tenant, error)
	FindByWhatsAppPhoneID(ctx context.Context, phoneID string) (*models.Tenant, error)
	FindByVerifyToken(ctx context.Context, token string) (*models.Tenant, error)
	Create(ctx context.Context, tenant *models.Tenant) error
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context) ([]models.Tenant, error)
}

type tenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) FindByID(ctx context.Context, id uint) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).First(&tenant, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByCNPJ(ctx context.Context, cnpj string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("cnpj = ?", cnpj).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant por CNPJ", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByTelefone(ctx context.Context, telefone string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("telefone = ?", telefone).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant por telefone", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByWhatsAppPhoneID(ctx context.Context, phoneID string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("whatsapp_phone_id = ?", phoneID).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant por WhatsApp phone ID", err)
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByVerifyToken(ctx context.Context, token string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("whatsapp_verify_token = ?", token).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("tenant não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar tenant por verify token", err)
	}
	return &tenant, nil
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

func (r *tenantRepository) List(ctx context.Context) ([]models.Tenant, error) {
	var tenants []models.Tenant
	if err := r.db.WithContext(ctx).Find(&tenants).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Tenant{}, nil
		}
		return nil, apperror.NewInternalError("falha ao listar tenants", err)
	}
	return tenants, nil
}
