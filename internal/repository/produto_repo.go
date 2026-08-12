package repository

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"gorm.io/gorm"
)

// ProdutoRepository defines the data access contract for products.
type ProdutoRepository interface {
	FindByID(ctx context.Context, id uint) (*models.Produto, error)
	FindByTenant(ctx context.Context, tenantID uint) ([]models.Produto, error)
	FindByTenantDisponiveis(ctx context.Context, tenantID uint) ([]models.Produto, error)
	FindByCategoria(ctx context.Context, tenantID, categoriaID uint) ([]models.Produto, error)
	FindByNome(ctx context.Context, tenantID uint, nome string) (*models.Produto, error)
	Create(ctx context.Context, produto *models.Produto) error
	Update(ctx context.Context, produto *models.Produto) error
	Delete(ctx context.Context, id uint) error
	BuscarProdutosPorNome(ctx context.Context, tenantID string, nome string, limit int) ([]dto.ProdutoItem, error)
	BuscarProdutosLote(ctx context.Context, tenantID string, nomes []string) (map[string]dto.ProdutoItem, error)
	FindWithFilters(ctx context.Context, tenantID uint, categoriaID *uint, disponivel *bool, nome string, limit, offset int) ([]models.Produto, int64, error)
}

type produtoRepository struct {
	db *gorm.DB
}

// NewProdutoRepository creates a product repository backed by the provided database.
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

func (r *produtoRepository) BuscarProdutosPorNome(ctx context.Context, tenantID string, nome string, limit int) ([]dto.ProdutoItem, error) {
	panic("implementar")
}

func (r *produtoRepository) BuscarProdutosLote(ctx context.Context, tenantID string, nomes []string) (map[string]dto.ProdutoItem, error) {
	panic("implementar")
}

// internal/repository/produto_repo.go
// Adicione estes métodos ao ProdutoRepository

// FindWithFilters busca produtos com filtros e paginação
func (r *produtoRepository) FindWithFilters(ctx context.Context, tenantID uint, categoriaID *uint, disponivel *bool, nome string, limit, offset int) ([]models.Produto, int64, error) {
	var produtos []models.Produto
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Produto{}).Where("tenant_id = ?", tenantID)

	if categoriaID != nil && *categoriaID > 0 {
		query = query.Where("categoria_id = ?", *categoriaID)
	}
	if disponivel != nil {
		query = query.Where("disponivel = ?", *disponivel)
	}
	if nome != "" {
		query = query.Where("nome LIKE ?", "%"+nome+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Categoria").
		Order("nome ASC").
		Limit(limit).
		Offset(offset).
		Find(&produtos).Error

	return produtos, total, err
}
