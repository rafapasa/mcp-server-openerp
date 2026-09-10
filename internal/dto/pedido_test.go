package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatarMensagemSaiuParaEntrega(t *testing.T) {
	pedido := &PedidoDTO{
		ID:          5,
		ClienteNome: "João",
		EnderecoEntrega: &EnderecoDTO{
			Logradouro: "Travessa Ana Albrecht",
			Numero:     "68",
			Bairro:     "Centro",
		},
	}
	msg := FormatarMensagemSaiuParaEntrega(pedido)
	assert.Contains(t, msg, "João")
	assert.Contains(t, msg, "#5")
	assert.Contains(t, msg, "Travessa Ana Albrecht")
	assert.Contains(t, msg, "68")
	assert.Contains(t, msg, "Centro")
}

func TestFormatarMensagemSaiuParaEntrega_SemNome(t *testing.T) {
	msg := FormatarMensagemSaiuParaEntrega(&PedidoDTO{ID: 7, ClienteNome: "   "})
	assert.Contains(t, msg, "Cliente")
	assert.Contains(t, msg, "#7")
	assert.NotContains(t, msg, "Endereço")
}

func TestFormatarMensagemSaiuParaEntrega_SemEndereco(t *testing.T) {
	msg := FormatarMensagemSaiuParaEntrega(&PedidoDTO{ID: 8, ClienteNome: "Maria"})
	assert.Contains(t, msg, "Maria")
	assert.NotContains(t, msg, "Endereço")
}

func TestFormatarMensagemSaiuParaEntrega_EnderecoParcial(t *testing.T) {
	msg := FormatarMensagemSaiuParaEntrega(&PedidoDTO{
		ID:              9,
		ClienteNome:     "Ana",
		EnderecoEntrega: &EnderecoDTO{Bairro: "Centro"},
	})
	assert.Contains(t, msg, "Endereço: Centro")
	assert.NotContains(t, msg, ", Centro")
}

func TestFormatarMensagemSaiuParaEntrega_Nil(t *testing.T) {
	assert.Equal(t, "", FormatarMensagemSaiuParaEntrega(nil))
}
