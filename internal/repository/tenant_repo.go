package repository

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"gorm.io/gorm"
)

// TenantRepository defines persistence operations for tenants.
type TenantRepository interface {
	FindByID(ctx context.Context, id uint) (*models.Tenant, error)
	FindByCNPJ(ctx context.Context, cnpj string) (*models.Tenant, error)
	FindByTelefone(ctx context.Context, telefone string) (*models.Tenant, error)
	FindByWhatsAppPhoneID(ctx context.Context, phoneID string) (*models.Tenant, error)
	FindByVerifyToken(ctx context.Context, token string) (*models.Tenant, error)
	Create(ctx context.Context, tenant *models.Tenant) error
	Update(ctx context.Context, tenant *models.Tenant) error
	List(ctx context.Context) ([]models.Tenant, error)
}

type tenantRepository struct {
	db *gorm.DB
}

// NewTenantRepository creates a repository for handling tenant persistence.
func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) FindByID(ctx context.Context, id uint) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).First(&tenant, id).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByCNPJ(ctx context.Context, cnpj string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("cnpj = ?", cnpj).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByTelefone(ctx context.Context, telefone string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("telefone = ?", telefone).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByWhatsAppPhoneID(ctx context.Context, phoneID string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("whatsapp_phone_id = ?", phoneID).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) FindByVerifyToken(ctx context.Context, token string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("whatsapp_verify_token = ?", token).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *tenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Save(tenant).Error
}

func (r *tenantRepository) List(ctx context.Context) ([]models.Tenant, error) {
	var tenants []models.Tenant
	if err := r.db.WithContext(ctx).Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}
