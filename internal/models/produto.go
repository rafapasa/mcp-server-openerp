package models

import "time"

type Produto struct {
	ID           uint      `gorm:"primaryKey"`
	TenantID     uint      `gorm:"not null;index"`
	CategoriaID  *uint     `gorm:"index"`
	Nome         string    `gorm:"size:100;not null"`
	Descricao    string    `gorm:"size:255"`
	Preco        float64   `gorm:"type:decimal(10,2);not null"`
	Ingredientes JSONArray `gorm:"type:json"`
	Disponivel   bool      `gorm:"default:true"`
	TempoPreparo int       `gorm:"default:15"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`

	Tenant    Tenant     `gorm:"foreignKey:TenantID"`
	Categoria *Categoria `gorm:"foreignKey:CategoriaID"`
}

func (Produto) TableName() string { return "produtos" }
