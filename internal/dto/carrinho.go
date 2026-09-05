// internal/dto/carrinho.go - EVOLUÍDO com estado de endereço
package dto

import "time"

// ItemCarrinho representa um item no carrinho
type ItemCarrinho struct {
	ProdutoItem ProdutoItem `json:"produto_item"`
	Quantidade  int         `json:"quantidade"`
	Observacao  string      `json:"observacao,omitempty"`
	Preco       float64     `json:"preco"`
}

// Estados do carrinho para fluxo de entrega
const (
	EstadoAberto                        = "aberto"
	EstadoAguardandoEnderecoLista       = "aguardando_endereco_selecao"
	EstadoAguardandoEnderecoNovo        = "aguardando_endereco_novo"
	EstadoAguardandoConfirmacaoEndereco = "aguardando_confirmacao_endereco"
	EstadoAguardandoPagamento           = "aguardando_pagamento"
	EstadoAguardandoValorPagamento      = "aguardando_valor_pagamento"
	EstadoAguardandoTroco               = "aguardando_troco"
)

// Carrinho representa o carrinho de um cliente - agora com máquina de estados de endereço
type Carrinho struct {
	ClienteID string         `json:"cliente_id"`
	TenantID  string         `json:"tenant_id"`
	Itens     []ItemCarrinho `json:"itens"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Fluxo de endereço de entrega - novo
	Estado                 string                 `json:"estado,omitempty"`              // aberto | aguardando_endereco_selecao | aguardando_endereco_novo
	EnderecoID             *uint                  `json:"endereco_id,omitempty"`         // ID do endereço escolhido para este pedido
	EnderecoTemporario     string                 `json:"endereco_temporario,omitempty"` // texto cru quando usuário digita novo endereço antes de cadastrar
	EnderecoPendente       *CriarEnderecoRequest  `json:"endereco_pendente,omitempty"`
	EnderecoConfirmacaoID  *uint                  `json:"endereco_confirmacao_id,omitempty"`
	EnderecoConfirmacaoIdx int                    `json:"endereco_confirmacao_idx,omitempty"`
	TentativasEndereco     int                    `json:"tentativas_endereco,omitempty"` // evita loop infinito
	Pagamentos             []PedidoPagamentoInput `json:"pagamentos,omitempty"`
	FormaPagamentoPendente uint                   `json:"forma_pagamento_pendente,omitempty"`
	ValorPagamentoPendente float64                `json:"valor_pagamento_pendente,omitempty"`
}
