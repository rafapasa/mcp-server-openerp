package repository

import (
	"context"
	"errors"

	"github.com/etoolstec/gokit/apperror"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
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
	BuscarProdutosPorNome(ctx context.Context, tenantID string, nome string, limit int) ([]dto.ProdutoItem, error)
	BuscarProdutosLote(ctx context.Context, tenantID string, nomes []string) (map[string]dto.ProdutoItem, error)
	FindWithFilters(ctx context.Context, tenantID uint, categoriaID *uint, disponivel *bool, nome string, limit, offset int) ([]models.Produto, int64, error)
	FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]models.Produto, int64, error)
}

type produtoRepository struct{ db *gorm.DB }

func NewProdutoRepository(db *gorm.DB) ProdutoRepository { return &produtoRepository{db: db} }

func (r *produtoRepository) FindByID(ctx context.Context, id uint) (*models.Produto, error) {
	var p models.Produto
	if err := r.db.WithContext(ctx).Preload("Categoria").Preload("Tenant").First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("produto não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar produto", err)
	}
	return &p, nil
}

func (r *produtoRepository) FindByTenant(ctx context.Context, tenantID uint) ([]models.Produto, error) {
	var list []models.Produto
	if err := r.db.WithContext(ctx).Preload("Categoria").Where("tenant_id = ?", tenantID).Order("categoria_id, nome").Find(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Produto{}, nil
		}
		return nil, apperror.NewInternalError("falha ao listar produtos", err)
	}
	return list, nil
}

func (r *produtoRepository) FindByTenantDisponiveis(ctx context.Context, tenantID uint) ([]models.Produto, error) {
	var list []models.Produto
	if err := r.db.WithContext(ctx).Preload("Categoria").Where("tenant_id = ? AND disponivel = ?", tenantID, true).Order("categoria_id, nome").Find(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Produto{}, nil
		}
		return nil, apperror.NewInternalError("falha ao listar produtos disponíveis", err)
	}
	return list, nil
}

func (r *produtoRepository) FindByCategoria(ctx context.Context, tenantID, categoriaID uint) ([]models.Produto, error) {
	var list []models.Produto
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND categoria_id = ? AND disponivel = ?", tenantID, categoriaID, true).Find(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Produto{}, nil
		}
		return nil, apperror.NewInternalError("falha ao buscar produtos por categoria", err)
	}
	return list, nil
}

func (r *produtoRepository) FindByNome(ctx context.Context, tenantID uint, nome string) (*models.Produto, error) {
	var p models.Produto
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND nome = ?", tenantID, nome).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("produto não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar produto por nome", err)
	}
	return &p, nil
}

func (r *produtoRepository) Create(ctx context.Context, produto *models.Produto) error {
	if err := r.db.WithContext(ctx).Create(produto).Error; err != nil {
		return apperror.NewInternalError("falha ao criar produto", err)
	}
	return nil
}

func (r *produtoRepository) Update(ctx context.Context, produto *models.Produto) error {
	if err := r.db.WithContext(ctx).Save(produto).Error; err != nil {
		return apperror.NewInternalError("falha ao atualizar produto", err)
	}
	return nil
}

func (r *produtoRepository) Delete(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&models.Produto{}, id)
	if res.Error != nil {
		return apperror.NewInternalError("falha ao deletar produto", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.NewNotFoundError("produto não encontrado")
	}
	return nil
}

func (r *produtoRepository) BuscarProdutosPorNome(ctx context.Context, tenantID string, nome string, limit int) ([]dto.ProdutoItem, error) {
	return nil, apperror.NewInternalError("BuscarProdutosPorNome não implementado", nil)
}

func (r *produtoRepository) BuscarProdutosLote(ctx context.Context, tenantID string, nomes []string) (map[string]dto.ProdutoItem, error) {
	return nil, apperror.NewInternalError("BuscarProdutosLote não implementado", nil)
}

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
		return nil, 0, apperror.NewInternalError("falha ao contar produtos", err)
	}
	if err := query.Preload("Categoria").Order("nome ASC").Limit(limit).Offset(offset).Find(&produtos).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Produto{}, total, nil
		}
		return nil, 0, apperror.NewInternalError("falha ao buscar produtos", err)
	}
	if produtos == nil {
		produtos = []models.Produto{}
	}
	return produtos, total, nil
}

func (r *produtoRepository) FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]models.Produto, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.Produto{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return nil, 0, apperror.NewInternalError("falha ao contar produtos", err)
	}
	var produtos []models.Produto
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Preload("Categoria").Order("nome ASC").Limit(limit).Offset(offset).Find(&produtos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Produto{}, total, nil
		}
		return nil, 0, apperror.NewInternalError("falha ao buscar produtos", err)
	}
	if produtos == nil {
		produtos = []models.Produto{}
	}
	return produtos, total, nil
}
