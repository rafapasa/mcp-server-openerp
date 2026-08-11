// internal/server/tools/helpers.go
package tools

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// GetArguments converte os arguments da request
func GetArguments(request mcp.CallToolRequest) (map[string]interface{}, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("arguments inválidos")
	}
	return args, nil
}

// GetString extrai uma string dos arguments
func GetString(args map[string]interface{}, key string) (string, bool) {
	val, ok := args[key].(string)
	return val, ok
}

// GetStringRequired extrai uma string obrigatória
func GetStringRequired(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key].(string)
	if !ok || val == "" {
		return "", fmt.Errorf("'%s' é obrigatório", key)
	}
	return val, nil
}

// GetItems extrai e valida a lista de itens
func GetItems(args map[string]interface{}) ([]service.ItemPedidoInput, error) {
	itensRaw, ok := args["itens"].([]interface{})
	if !ok || len(itensRaw) == 0 {
		return nil, fmt.Errorf("itens é obrigatório")
	}

	var itens []service.ItemPedidoInput
	for _, itemRaw := range itensRaw {
		itemMap, ok := itemRaw.(map[string]interface{})
		if !ok {
			continue
		}

		nome, _ := itemMap["nome"].(string)
		qtd, _ := itemMap["quantidade"].(float64)
		obs, _ := itemMap["observacao"].(string)

		if nome != "" && qtd > 0 {
			itens = append(itens, service.ItemPedidoInput{
				Nome:       nome,
				Quantidade: int(qtd),
				Observacao: obs,
			})
		}
	}

	if len(itens) == 0 {
		return nil, fmt.Errorf("nenhum item válido encontrado")
	}

	return itens, nil
}

// FormatResumoCarrinho formata a mensagem de resumo do carrinho
func FormatResumoCarrinho(itens []service.ItemCarrinho, total float64, tempoEstimado int) string {
	var sb strings.Builder
	sb.WriteString("🛒 **SEU CARRINHO**\n\n")

	if len(itens) == 0 {
		sb.WriteString("Seu carrinho está vazio. Adicione itens usando: *quero um X-Bacon*")
		return sb.String()
	}

	for _, item := range itens {
		totalItem := item.Preco * float64(item.Quantidade)
		sb.WriteString(fmt.Sprintf("• %dx **%s** - R$ %.2f\n",
			item.Quantidade, item.Nome, totalItem))
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
func FormatRespostaPedido(pedido *service.PedidoConfirmado) string {
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
			item.Quantidade, item.Nome, totalItem))
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
