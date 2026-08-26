package helpers

import (
	"fmt"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

// FormatResumoCarrinho formata a mensagem de resumo do carrinho
func FormatResumoCarrinho(itens []dto.ItemCarrinho, total float64, tempoEstimado int) string {
	var sb strings.Builder
	sb.WriteString("🛒 **SEU CARRINHO**\n\n")

	if len(itens) == 0 {
		sb.WriteString("Seu carrinho está vazio. Adicione itens usando: *quero um X-Bacon*")
		return sb.String()
	}

	for _, item := range itens {
		totalItem := item.Preco * float64(item.Quantidade)
		sb.WriteString(fmt.Sprintf("• %dx **%s** - R$ %.2f\n",
			item.Quantidade, item.ProdutoItem.Nome, totalItem))
		if item.Observacao != "" {
			sb.WriteString(fmt.Sprintf("  _Obs: %s_\n", item.Observacao))
		}
	}

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("💰 **Total: R$ %.2f**\n", total))
	if tempoEstimado > 0 {
		sb.WriteString(fmt.Sprintf("⏱️ **Tempo estimado:** %d minutos\n", tempoEstimado))
	}
	sb.WriteString("\n📝 *Comandos:*\n")
	sb.WriteString("• Adicionar: *quero mais um X-Bacon*\n")
	sb.WriteString("• Remover: *remover Coca-Cola* ou *remover 2 Coca-Cola*\n")
	sb.WriteString("• Finalizar: *finalizar pedido* ou *confirmar*\n")
	sb.WriteString("• Limpar: *limpar carrinho*")

	return sb.String()
}

// FormatRespostaPedido monta a mensagem de confirmação
func FormatRespostaPedido(pedido *dto.PedidoConfirmado) string {
	var sb strings.Builder
	sb.WriteString("✅ **PEDIDO CONFIRMADO!**\n\n")
	sb.WriteString(fmt.Sprintf("🧾 **Pedido #%d**\n", pedido.ID))
	if pedido.ClienteNome != "" {
		sb.WriteString(fmt.Sprintf("👤 **Cliente:** %s\n", pedido.ClienteNome))
	}
	sb.WriteString("---\n")

	for _, item := range pedido.Itens {
		totalItem := item.PrecoUnitario * float64(item.Quantidade)
		sb.WriteString(fmt.Sprintf("• %dx **%s** - R$ %.2f\n",
			item.Quantidade, item.ProdutoItem.Nome, totalItem))
		if item.Observacao != "" {
			sb.WriteString(fmt.Sprintf("  _Obs: %s_\n", item.Observacao))
		}
	}

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("💰 **Total: R$ %.2f**\n", pedido.Total))
	sb.WriteString(fmt.Sprintf("⏱️ **Tempo estimado:** %d minutos\n", pedido.TempoEstimado))
	sb.WriteString("\n🙏 Obrigado pela preferência! Volte sempre! 🍔")

	return sb.String()
}
