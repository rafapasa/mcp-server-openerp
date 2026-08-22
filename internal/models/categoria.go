package models

import "time"

type Categoria struct {
	ID        uint      `gorm:"primaryKey"`
	TenantID  uint      `gorm:"not null;index"`
	Nome      string    `gorm:"size:50;not null"`
	Ordem     int       `gorm:"default:0"`
	Ativo     bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	Tenant   Tenant    `gorm:"foreignKey:TenantID"`
	Produtos []Produto `gorm:"foreignKey:CategoriaID"`
}

func (Categoria) TableName() string { return "categorias" }
