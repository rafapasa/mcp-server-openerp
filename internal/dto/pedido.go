// internal/dto/pedido.go - MERGE de pedido.go + pedido_dto.go
// Junta os dois arquivos, mescla campos duplicados sem perder nada
package dto

import (
	"fmt"
	"strings"
	"time"
)

// ============================================
// ITENS - ENTRADA DA IA
// ============================================

type ItemPedidoInput struct {
	ProdutoItem   ProdutoItem `json:"produto_item"`
	Quantidade    int         `json:"quantidade"`
	Observacao    string      `json:"observacao"`
	PrecoUnitario float64     `json:"preco_unitario,omitempty"`
}

type PedidoExtraido struct {
	Itens       []ItemPedidoInput `json:"itens"`
	Bebidas     []ItemPedidoInput `json:"bebidas,omitempty"`
	Observacoes string            `json:"observacoes,omitempty"`
}

// ============================================
// PEDIDO CONFIRMADO - RETORNO PARA WHATSAPP
// ============================================

type PedidoConfirmado struct {
	ID              int                  `json:"id"`
	TenantID        string               `json:"tenant_id"`
	ClienteID       string               `json:"cliente_id"`
	ClienteNome     string               `json:"cliente_nome"`
	Itens           []ItemPedidoInput    `json:"itens"`
	Total           float64              `json:"total"`
	TempoEstimado   int                  `json:"tempo_estimado"`
	Status          string               `json:"status"`
	CriadoEm        string               `json:"criado_em"`
	EnderecoEntrega *EnderecoDTO         `json:"endereco_entrega,omitempty"`
	Pagamentos      []PedidoPagamentoDTO `json:"pagamentos,omitempty"`
}

// ============================================
// DTOs DE API / DASHBOARD - MERGE DOS DOIS ARQUIVOS
// ============================================

type PedidoDTO struct {
	ID                uint                 `json:"id"`
	TenantID          uint                 `json:"tenant_id"`
	ClienteID         *uint                `json:"cliente_id"`
	ClienteNome       string               `json:"cliente_nome"`
	ClienteTelefone   string               `json:"cliente_telefone"`
	EnderecoEntregaID *uint                `json:"endereco_entrega_id"`
	EnderecoEntrega   *EnderecoDTO         `json:"endereco_entrega,omitempty"`
	Itens             []ItemPedidoDTO      `json:"itens"`
	Total             float64              `json:"total"`
	Status            string               `json:"status"`
	Observacoes       string               `json:"observacoes"`
	TempoEstimado     int                  `json:"tempo_estimado"`
	Origem            string               `json:"origem"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
	Pagamentos        []PedidoPagamentoDTO `json:"pagamentos,omitempty"`
}

// FormatarMensagemSaiuParaEntrega monta a mensagem enviada ao cliente quando o
// pedido entra em `saiu_para_entrega`. Fica no dto para ser compartilhada entre
// o service e a borda HTTP sem criar dependência circular.
func FormatarMensagemSaiuParaEntrega(pedido *PedidoDTO) string {
	if pedido == nil {
		return ""
	}
	clienteNome := strings.TrimSpace(pedido.ClienteNome)
	if clienteNome == "" {
		clienteNome = "Cliente"
	}
	msg := fmt.Sprintf("🛵 %s, seu pedido #%d saiu para entrega! Chega em ~15 min.", clienteNome, pedido.ID)
	if pedido.EnderecoEntrega == nil {
		return msg
	}

	endereco := strings.TrimSpace(pedido.EnderecoEntrega.Logradouro)
	if pedido.EnderecoEntrega.Numero != "" {
		if endereco != "" {
			endereco += ", " + pedido.EnderecoEntrega.Numero
		} else {
			endereco = pedido.EnderecoEntrega.Numero
		}
	}
	if pedido.EnderecoEntrega.Bairro != "" {
		if endereco != "" {
			endereco += " - " + pedido.EnderecoEntrega.Bairro
		} else {
			endereco = pedido.EnderecoEntrega.Bairro
		}
	}
	if endereco != "" {
		msg += fmt.Sprintf(" Endereço: %s", endereco)
	}
	return msg
}

// ItemPedidoDTO - MERGE: tinha versão com ProdutoID e versão sem
type ItemPedidoDTO struct {
	ProdutoID  *uint   `json:"produto_id,omitempty"` // merge: opcional para compatibilidade
	Nome       string  `json:"nome"`
	Quantidade int     `json:"quantidade"`
	Preco      float64 `json:"preco"`
	Observacao string  `json:"observacao,omitempty"`
}

// CriarPedidoRequest - MERGE dos dois arquivos
type CriarPedidoRequest struct {
	TenantID          uint            `json:"tenant_id"`
	ClienteID         *uint           `json:"cliente_id"`
	EnderecoEntregaID *uint           `json:"endereco_entrega_id"`
	Itens             []ItemPedidoDTO `json:"itens"`
	Observacoes       string          `json:"observacoes,omitempty"`
	Origem            string          `json:"origem,omitempty"`
}

func (r *CriarPedidoRequest) Total() float32 {
	var total float64
	for _, item := range r.Itens {
		total += float64(item.Quantidade) * item.Preco
	}
	return float32(total)
}

func (r *CriarPedidoRequest) TotalFloat64() float64 {
	var total float64
	for _, item := range r.Itens {
		total += float64(item.Quantidade) * item.Preco
	}
	return total
}

type PedidoListResponseDTO struct {
	Pedidos    []PedidoDTO `json:"pedidos"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int64       `json:"total_pages"`
}
