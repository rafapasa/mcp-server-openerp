package repository

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"gorm.io/gorm"
)

type ProdutoRepository interface {
	FindByID(ctx context.Context, id uint) (*models.Produto, error)
	FindByTenant(ctx context.Context, tenantID uint) ([]models.Produto, error)
	FindByTenantDisponiveis(ctx context.Context, tenantID uint) ([]models.Produto, error)
	FindByCategoria(ctx context.Context, tenantID, categoriaID uint) ([]models.Produto, error)
	FindByNome(ctx context.Context, tenantID uint, nome string) (*models.Produto, error)
	Create(ctx context.Context, produto *models.Produto) error
	Update(ctx context.Context, produto *models.Produto) error
	Delete(ctx context.Context, id uint) error
}

type produtoRepository struct {
	db *gorm.DB
}

func NewProdutoRepository(db *gorm.DB) ProdutoRepository {
	return &produtoRepository{db: db}
}

func (r *produtoRepository) FindByID(ctx context.Context, id uint) (*models.Produto, error) {
	var produto models.Produto
	if err := r.db.WithContext(ctx).
		Preload("Categoria").
		Preload("Tenant").
		First(&produto, id).Error; err != nil {
		return nil, err
	}
	return &produto, nil
}

func (r *produtoRepository) FindByTenant(ctx context.Context, tenantID uint) ([]models.Produto, error) {
	var produtos []models.Produto
	if err := r.db.WithContext(ctx).
		Preload("Categoria").
		Where("tenant_id = ?", tenantID).
		Order("categoria_id, nome").
		Find(&produtos).Error; err != nil {
		return nil, err
	}
	return produtos, nil
}

func (r *produtoRepository) FindByTenantDisponiveis(ctx context.Context, tenantID uint) ([]models.Produto, error) {
	var produtos []models.Produto
	if err := r.db.WithContext(ctx).
		Preload("Categoria").
		Where("tenant_id = ? AND disponivel = ?", tenantID, true).
		Order("categoria_id, nome").
		Find(&produtos).Error; err != nil {
		return nil, err
	}
	return produtos, nil
}

func (r *produtoRepository) FindByCategoria(ctx context.Context, tenantID, categoriaID uint) ([]models.Produto, error) {
	var produtos []models.Produto
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND categoria_id = ? AND disponivel = ?", tenantID, categoriaID, true).
		Find(&produtos).Error; err != nil {
		return nil, err
	}
	return produtos, nil
}

func (r *produtoRepository) FindByNome(ctx context.Context, tenantID uint, nome string) (*models.Produto, error) {
	var produto models.Produto
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND nome = ?", tenantID, nome).
		First(&produto).Error; err != nil {
		return nil, err
	}
	return &produto, nil
}

func (r *produtoRepository) Create(ctx context.Context, produto *models.Produto) error {
	return r.db.WithContext(ctx).Create(produto).Error
}

func (r *produtoRepository) Update(ctx context.Context, produto *models.Produto) error {
	return r.db.WithContext(ctx).Save(produto).Error
}

func (r *produtoRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Produto{}, id).Error
}
