package repository

import (
	"context"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"gorm.io/gorm"
)

type UserRepositoryInterface interface {
	FindByEmail(ctx context.Context, tenantID uint, email string) (*models.User, error)
	FindByID(ctx context.Context, id uint) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
}

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) UserRepositoryInterface {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(ctx context.Context, tenantID uint, email string) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND email = ? AND is_active = true", tenantID, email).First(&u).Error
	return &u, err
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	return &u, err
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}
