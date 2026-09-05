package helpers

import (
	"fmt"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

func FormatConfirmacaoEndereco(endereco *dto.CriarEnderecoRequest) string {
	complemento := ""
	if endereco.Complemento != "" {
		complemento = " - " + endereco.Complemento
	}
	localidade := strings.Trim(strings.Join([]string{endereco.Bairro, endereco.Cidade}, ", "), ", ")
	if localidade != "" {
		localidade = " - " + localidade
	}
	return fmt.Sprintf("📍 **Confirma entrega em:** %s, %s%s%s - %s?\n\nResponda *sim* ou *corrigir*.",
		endereco.Logradouro, endereco.Numero, complemento, localidade, endereco.Estado)
}

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

// FormatRespostaPedido monta a mensagem de confirmação - CORRIGIDO para usar campos que existem
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

	// Mostra endereço se tiver - usa DTO que agora existe
	if pedido.EnderecoEntrega != nil && pedido.EnderecoEntrega.Logradouro != "" {
		sb.WriteString(fmt.Sprintf("📍 **Entrega em:** %s, %s\n", pedido.EnderecoEntrega.Logradouro, pedido.EnderecoEntrega.Numero))
		if pedido.EnderecoEntrega.Bairro != "" {
			sb.WriteString(fmt.Sprintf("   %s", pedido.EnderecoEntrega.Bairro))
			if pedido.EnderecoEntrega.Cidade != "" {
				sb.WriteString(fmt.Sprintf(" - %s/%s", pedido.EnderecoEntrega.Cidade, pedido.EnderecoEntrega.Estado))
			}
			sb.WriteString("\n")
		}
		if pedido.EnderecoEntrega.Referencia != "" {
			sb.WriteString(fmt.Sprintf("   _Ref: %s_\n", pedido.EnderecoEntrega.Referencia))
		}
	}

	sb.WriteString("\n🙏 Obrigado pela preferência! Volte sempre! 🍔")

	return sb.String()
}

// ============================================
// NOVOS - FLUXO DE ENDEREÇO
// ============================================

func FormatListaEnderecos(enderecos []dto.EnderecoDTO) string {
	var sb strings.Builder
	sb.WriteString("📍 **SEUS ENDEREÇOS CADASTRADOS**\n\n")
	sb.WriteString("Você já tem endereços salvos. Qual usar para entrega?\n\n")

	for i, e := range enderecos {
		num := i + 1
		princ := ""
		if e.Principal {
			princ = " ⭐ _Principal_"
		}
		sb.WriteString(fmt.Sprintf("*%d* - %s, %s", num, e.Logradouro, e.Numero))
		if e.Complemento != "" {
			sb.WriteString(fmt.Sprintf(" %s", e.Complemento))
		}
		sb.WriteString(princ + "\n")
		if e.Bairro != "" || e.Cidade != "" {
			sb.WriteString(fmt.Sprintf("   %s", e.Bairro))
			if e.Cidade != "" {
				sb.WriteString(fmt.Sprintf(" - %s/%s", e.Cidade, e.Estado))
			}
			sb.WriteString("\n")
		}
		if e.Referencia != "" {
			sb.WriteString(fmt.Sprintf("   _Ref: %s_\n", e.Referencia))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("Digite o *número* do endereço (ex: *1*) ou\n")
	sb.WriteString("digite *novo* para cadastrar um novo endereço 📝")

	return sb.String()
}

func FormatSolicitarNovoEndereco(temEnderecos bool) string {
	var sb strings.Builder
	if !temEnderecos {
		sb.WriteString("📍 **ENDEREÇO DE ENTREGA**\n\n")
		sb.WriteString("Vi que você ainda não tem endereço cadastrado.\n\n")
	} else {
		sb.WriteString("📍 **NOVO ENDEREÇO**\n\n")
	}

	sb.WriteString("Me envie seu endereço completo, por favor:\n\n")
	sb.WriteString("Exemplo:\n")
	sb.WriteString("*Rua das Flores, 123, Centro, Pinhalzinho - SC, 89870-000*\n\n")
	sb.WriteString("Pode incluir:\n")
	sb.WriteString("• Rua e número\n")
	sb.WriteString("• Bairro\n")
	sb.WriteString("• Ponto de referência (opcional)\n\n")
	sb.WriteString("_Vou salvar para seus próximos pedidos!_")

	return sb.String()
}

func FormatEnderecoCadastrado(endereco *dto.EnderecoDTO) string {
	var sb strings.Builder
	sb.WriteString("✅ **Endereço cadastrado!**\n\n")
	sb.WriteString(fmt.Sprintf("📍 %s, %s\n", endereco.Logradouro, endereco.Numero))
	if endereco.Bairro != "" {
		sb.WriteString(fmt.Sprintf("   %s - %s/%s\n", endereco.Bairro, endereco.Cidade, endereco.Estado))
	}
	if endereco.CEP != "" {
		sb.WriteString(fmt.Sprintf("   CEP: %s\n", endereco.CEP))
	}
	sb.WriteString("\n_Já vou usar ele para este pedido..._")
	return sb.String()
}

func FormatErroEndereco(msg string) string {
	return fmt.Sprintf("⚠️ %s\n\n%s", msg, FormatSolicitarNovoEndereco(true))
}

func FormatListaFormasPagamento(formas []dto.FormaPagamentoDTO) string {
	var sb strings.Builder
	sb.WriteString("💳 **COMO VAI PAGAR?**\n\n")
	for i, forma := range formas {
		sb.WriteString(fmt.Sprintf("*%d* - %s\n", i+1, forma.Nome))
	}
	sb.WriteString("\nVocê pode escolher mais de uma forma de pagamento.")
	return sb.String()
}
