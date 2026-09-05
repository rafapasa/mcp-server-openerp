package models

import "time"

const (
	TipoPagamentoDinheiro      = "dinheiro"
	TipoPagamentoPix           = "pix"
	TipoPagamentoCartaoCredito = "cartao_credito"
	TipoPagamentoCartaoDebito  = "cartao_debito"
)

type FormaPagamento struct {
	ID        uint      `gorm:"primaryKey"`
	TenantID  uint      `gorm:"not null;index"`
	Nome      string    `gorm:"size:100;not null"`
	Tipo      string    `gorm:"type:enum('dinheiro','pix','cartao_credito','cartao_debito');not null"`
	Ativo     bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	Tenant Tenant `gorm:"foreignKey:TenantID"`
}

func (FormaPagamento) TableName() string { return "formas_pagamento" }

const (
	StatusPagamentoPendente = "pendente"
	StatusPagamentoPago     = "pago"
)

type PedidoPagamento struct {
	ID               uint      `gorm:"primaryKey"`
	PedidoID         uint      `gorm:"not null;index"`
	FormaPagamentoID uint      `gorm:"not null;index"`
	Valor            float64   `gorm:"type:decimal(10,2);not null"`
	TrocoPara        *float64  `gorm:"type:decimal(10,2)"`
	Status           string    `gorm:"type:enum('pendente','pago');not null;default:'pendente'"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`

	Pedido         Pedido         `gorm:"foreignKey:PedidoID"`
	FormaPagamento FormaPagamento `gorm:"foreignKey:FormaPagamentoID"`
}

func (PedidoPagamento) TableName() string { return "pedido_pagamentos" }
