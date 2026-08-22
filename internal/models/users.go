package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	TenantID     uint      `gorm:"column:tenant_id;not null;index;uniqueIndex:uq_tenant_email"`
	Nome         string    `gorm:"size:100;not null"`
	Email        string    `gorm:"size:255;not null;uniqueIndex:uq_tenant_email"`
	PasswordHash string    `gorm:"column:password_hash;size:255;not null"`
	Role         string    `gorm:"type:enum('admin','atendente','cozinha');default:admin"`
	IsActive     bool      `gorm:"column:is_active;default:true"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (User) TableName() string { return "users" }
