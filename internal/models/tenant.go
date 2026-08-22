package models

import "time"

type Tenant struct {
	ID        uint      `gorm:"primaryKey"`
	Nome      string    `gorm:"size:100;not null"`
	CNPJ      string    `gorm:"size:18;uniqueIndex"`
	Telefone  string    `gorm:"size:20"`
	Endereco  string    `gorm:"size:255"`
	Segmento  string    `gorm:"size:50;default:'geral'"` // farmacia, autopecas, restaurante
	Ativo     bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	Categorias []Categoria `gorm:"foreignKey:TenantID"`
	Produtos   []Produto   `gorm:"foreignKey:TenantID"`
	Pedidos    []Pedido    `gorm:"foreignKey:TenantID"`
	Clientes   []Cliente   `gorm:"foreignKey:TenantID"`
}

func (Tenant) TableName() string { return "tenants" }
