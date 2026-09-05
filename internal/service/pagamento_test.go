package service

import (
	"context"
	"testing"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/require"
)

type pagamentoFormaRepoFake struct {
	forma *models.FormaPagamento
}

func (f *pagamentoFormaRepoFake) FindByID(context.Context, uint) (*models.FormaPagamento, error) {
	return f.forma, nil
}
func (f *pagamentoFormaRepoFake) FindByTenant(context.Context, uint, bool) ([]models.FormaPagamento, error) {
	return nil, nil
}
func (f *pagamentoFormaRepoFake) Create(context.Context, *models.FormaPagamento) error { return nil }
func (f *pagamentoFormaRepoFake) Update(context.Context, *models.FormaPagamento) error { return nil }
func (f *pagamentoFormaRepoFake) Delete(context.Context, uint, uint) error             { return nil }

type pagamentoRepoFake struct{}

func (pagamentoRepoFake) FindByPedido(context.Context, uint) ([]models.PedidoPagamento, error) {
	return nil, nil
}
func (pagamentoRepoFake) CreateMany(context.Context, []models.PedidoPagamento) error { return nil }
func (pagamentoRepoFake) MarcarPendentesComoPagos(context.Context, uint) error       { return nil }

func TestValidarFormaPagamento(t *testing.T) {
	_, tipo, err := validarFormaPagamento("  Pix  ", models.TipoPagamentoPix)
	require.NoError(t, err)
	require.Equal(t, models.TipoPagamentoPix, tipo)

	_, _, err = validarFormaPagamento("", models.TipoPagamentoPix)
	require.Error(t, err)
	_, _, err = validarFormaPagamento("Dinheiro", "boleto")
	require.Error(t, err)
}

func TestPedidoServicePrepararPagamentos(t *testing.T) {
	service := &PedidoService{
		formaPagamentoRepo: &pagamentoFormaRepoFake{
			forma: &models.FormaPagamento{
				ID: 1, TenantID: 10, Tipo: models.TipoPagamentoDinheiro, Ativo: true,
			},
		},
		pagamentoRepo: pagamentoRepoFake{},
	}
	troco := 50.0

	pagamentos, err := service.prepararPagamentos(context.Background(), 10, 40, []dto.PedidoPagamentoInput{
		{FormaPagamentoID: 1, Valor: 40, TrocoPara: &troco},
	})
	require.NoError(t, err)
	require.Len(t, pagamentos, 1)
	require.Equal(t, models.StatusPagamentoPendente, pagamentos[0].Status)

	_, err = service.prepararPagamentos(context.Background(), 10, 40, []dto.PedidoPagamentoInput{
		{FormaPagamentoID: 1, Valor: 20},
	})
	require.Error(t, err)

	service.formaPagamentoRepo.(*pagamentoFormaRepoFake).forma.Tipo = models.TipoPagamentoPix
	_, err = service.prepararPagamentos(context.Background(), 10, 40, []dto.PedidoPagamentoInput{
		{FormaPagamentoID: 1, Valor: 40, TrocoPara: &troco},
	})
	require.Error(t, err)
}
