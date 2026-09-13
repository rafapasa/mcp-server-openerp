package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/etoolstec/gokit/apperror"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
)

type UserRepositoryInterface interface {
	FindByEmail(ctx context.Context, tenantID uint, email string) (*[]models.User, error)
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]models.User, int64, error)
	Create(ctx context.Context, user *models.User) error
}

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) UserRepositoryInterface {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(ctx context.Context, tenantID uint, email string) (*[]models.User, error) {
	var u []models.User
	query := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ? AND is_active = 1", email)
	var err error
	if tenantID == 0 {
		err = query.Find(&u).Error
	} else {
		err = query.Where("tenant_id = ?", tenantID).Find(&u).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			empty := []models.User{}
			return &empty, nil
		}
		return nil, apperror.NewInternalError("falha ao buscar usuário por email", err)
	}
	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("usuário não encontrado")
		}
		return nil, apperror.NewInternalError("falha ao buscar usuário", err)
	}
	return &u, nil
}

func (r *userRepository) FindByTenantPaginated(ctx context.Context, tenantID uint, page int, limit int) ([]models.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("tenant_id = ?", tenantID).
		Count(&total).Error; err != nil {
		return nil, 0, apperror.NewInternalError("falha ao contar usuários", err)
	}

	var users []models.User
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.User{}, total, nil
		}
		return nil, 0, apperror.NewInternalError("falha ao buscar usuários", err)
	}
	if users == nil {
		users = []models.User{}
	}
	return users, total, nil
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return apperror.NewInternalError("falha ao criar usuário", err)
	}
	return nil
}
