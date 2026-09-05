package dto

type FormaPagamentoDTO struct {
	ID       uint   `json:"id"`
	TenantID uint   `json:"tenant_id"`
	Nome     string `json:"nome"`
	Tipo     string `json:"tipo"`
	Ativo    bool   `json:"ativo"`
}

type CriarFormaPagamentoRequest struct {
	Nome string `json:"nome"`
	Tipo string `json:"tipo"`
}

type AtualizarFormaPagamentoRequest struct {
	Nome  string `json:"nome"`
	Tipo  string `json:"tipo"`
	Ativo *bool  `json:"ativo"`
}

type PedidoPagamentoDTO struct {
	ID               uint               `json:"id"`
	PedidoID         uint               `json:"pedido_id"`
	FormaPagamentoID uint               `json:"forma_pagamento_id"`
	FormaPagamento   *FormaPagamentoDTO `json:"forma_pagamento,omitempty"`
	Valor            float64            `json:"valor"`
	TrocoPara        *float64           `json:"troco_para,omitempty"`
	Status           string             `json:"status"`
}

type PedidoPagamentoInput struct {
	FormaPagamentoID uint     `json:"forma_pagamento_id"`
	Valor            float64  `json:"valor"`
	TrocoPara        *float64 `json:"troco_para,omitempty"`
}
