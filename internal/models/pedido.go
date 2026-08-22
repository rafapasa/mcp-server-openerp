package models

import (
	"encoding/json"
	"time"
)

type ItemPedido struct {
	ProdutoID  uint    `json:"produto_id"`
	Nome       string  `json:"nome"`
	Quantidade int     `json:"qtd"`
	Preco      float64 `json:"preco"`
	Observacao string  `json:"obs,omitempty"`
}

type Pedido struct {
	ID                uint            `gorm:"primaryKey"`
	TenantID          uint            `gorm:"not null;index"`
	ClienteID         *uint           `gorm:"index:idx_pedidos_cliente_id"`
	EnderecoEntregaID *uint           `gorm:"index:idx_pedidos_endereco_entrega"`
	ClienteIDExterno  string          `gorm:"type:varchar(50);index"`
	ClienteNome       string          `gorm:"size:100"`
	ClienteTelefone   string          `gorm:"size:20"`
	Itens             json.RawMessage `gorm:"type:json;not null"`
	Total             float64         `gorm:"type:decimal(10,2);not null"`
	Status            string          `gorm:"type:enum('pendente','confirmado','preparando','entregue','cancelado');default:'pendente'"`
	Observacoes       string          `gorm:"type:text"`
	TempoEstimado     int             `gorm:"default:0"`
	Origem            string          `gorm:"type:enum('whatsapp','dashboard','api');default:'whatsapp'"`
	CreatedAt         time.Time       `gorm:"autoCreateTime;index:idx_pedidos_tenant_created,priority:2"`
	UpdatedAt         time.Time       `gorm:"autoUpdateTime"`

	Tenant          Tenant    `gorm:"foreignKey:TenantID"`
	Cliente         *Cliente  `gorm:"foreignKey:ClienteID"`
	EnderecoEntrega *Endereco `gorm:"foreignKey:EnderecoEntregaID"`
}

func (Pedido) TableName() string { return "pedidos" }
