package service

import (
	"testing"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestPedidoStatusValido(t *testing.T) {
	assert.True(t, pedidoStatusValido(models.StatusConfirmado, models.StatusEmPreparo))
	assert.True(t, pedidoStatusValido(models.StatusEmPreparo, models.StatusSaiuParaEntrega))
	assert.True(t, pedidoStatusValido(models.StatusSaiuParaEntrega, models.StatusEntregue))
	assert.True(t, pedidoStatusValido(models.StatusConfirmado, models.StatusCancelado))
	assert.False(t, pedidoStatusValido(models.StatusConfirmado, models.StatusEntregue))
	assert.False(t, pedidoStatusValido(models.StatusEntregue, models.StatusSaiuParaEntrega))
}

func TestNormalizarStatusPedido(t *testing.T) {
	assert.Equal(t, models.StatusEmPreparo, models.NormalizarStatusPedido("preparando"))
	assert.Equal(t, models.StatusSaiuParaEntrega, models.NormalizarStatusPedido("saiu para entrega"))
	assert.Equal(t, models.StatusSaiuParaEntrega, models.NormalizarStatusPedido("saiu_para_entrega"))
}

func TestFormatarMensagemSaiuParaEntrega(t *testing.T) {
	pedido := &models.Pedido{
		ID:          5,
		ClienteNome: "João",
		EnderecoEntrega: &models.Endereco{
			Logradouro: "Travessa Ana Albrecht",
			Numero:     "68",
			Bairro:     "Centro",
		},
	}
	msg := formatarMensagemSaiuParaEntrega(pedido)
	assert.Contains(t, msg, "João")
	assert.Contains(t, msg, "#5")
	assert.Contains(t, msg, "Travessa Ana Albrecht")
	assert.Contains(t, msg, "68")
}
