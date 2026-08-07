// internal/server/tools.go
package server

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rafapasa/mcp-server-openerp/internal/service"
)

// registerTools registra todas as ferramentas do servidor
func (s *MCPServer) registerTools() {
	s.AddTool(whatsappTool(), s.whatsappHandler())
	s.AddTool(processarPedidoTool(), s.processarPedidoHandler())
	s.AddTool(consultarCardapioTool(), s.consultarCardapioHandler())
}

// ============================================
// TOOL 1: processar_mensagem_whatsapp
// ============================================

func whatsappTool() mcp.Tool {
	return mcp.NewTool("processar_mensagem_whatsapp",
		mcp.WithDescription("Recebe uma mensagem de WhatsApp, extrai itens do pedido usando IA e processa"),
		mcp.WithString("mensagem",
			mcp.Required(),
			mcp.Description("Mensagem enviada pelo cliente no WhatsApp"),
		),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do restaurante (tenant)"),
		),
		mcp.WithString("cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente no WhatsApp (número de telefone)"),
		),
		mcp.WithString("cliente_nome",
			mcp.Description("Nome do cliente (opcional)"),
		),
	)
}

func (s *MCPServer) whatsappHandler() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Converte argumentos
		args, err := getArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Extrai parâmetros
		mensagem, _ := getString(args, "mensagem")
		if mensagem == "" {
			return mcp.NewToolResultError("mensagem é obrigatória"), nil
		}

		tenantID, err := getStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := getStringRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteNome, _ := getString(args, "cliente_nome")

		log.Printf("[WhatsApp] Processando mensagem de %s (%s): %s", clienteID, tenantID, mensagem)

		// Busca cardápio
		cardapio, err := s.getCardapio(tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}

		if len(cardapio) == 0 {
			return mcp.NewToolResultError("Cardápio não encontrado para este restaurante"), nil
		}

		// Extrai pedido com IA
		pedidoExtraido, err := s.extractOrderWithLLM(mensagem, cardapio)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao interpretar mensagem: %v", err)), nil
		}

		if len(pedidoExtraido.Itens) == 0 && len(pedidoExtraido.Bebidas) == 0 {
			return mcp.NewToolResultError(
				"Não foi possível identificar itens do pedido. Por favor, especifique os produtos desejados.",
			), nil
		}

		// Processa pedido
		pedidoConfirmado, err := s.processarPedido(tenantID, clienteID, clienteNome, pedidoExtraido)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao processar pedido: %v", err)), nil
		}

		// Monta resposta
		resposta := s.formatarRespostaPedido(pedidoConfirmado)

		return mcp.NewToolResultText(resposta), nil
	}
}

// ============================================
// TOOL 2: processar_pedido_restaurante
// ============================================

func processarPedidoTool() mcp.Tool {
	return mcp.NewTool("processar_pedido_restaurante",
		mcp.WithDescription("Processa um pedido manualmente (usado pelo dashboard do restaurante)"),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do restaurante"),
		),
		mcp.WithString("cliente_id",
			mcp.Required(),
			mcp.Description("ID do cliente"),
		),
		mcp.WithString("cliente_nome",
			mcp.Description("Nome do cliente"),
		),
		mcp.WithArray("itens",
			mcp.Required(),
			mcp.Description("Lista de itens do pedido"),
		),
		mcp.WithString("observacoes",
			mcp.Description("Observações gerais do pedido"),
		),
	)
}

func (s *MCPServer) processarPedidoHandler() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := getArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := getStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteID, err := getStringRequired(args, "cliente_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Extrai itens
		itens, err := getItems(args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		clienteNome, _ := getString(args, "cliente_nome")
		observacoes, _ := getString(args, "observacoes")

		// Processa pedido
		pedidoExtraido := &service.PedidoExtraido{
			Itens:       itens,
			Observacoes: observacoes,
		}

		pedidoConfirmado, err := s.processarPedido(tenantID, clienteID, clienteNome, pedidoExtraido)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao processar pedido: %v", err)), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("✅ Pedido #%d processado com sucesso!\nTotal: R$ %.2f",
				pedidoConfirmado.ID, pedidoConfirmado.Total),
		), nil
	}
}

// ============================================
// TOOL 3: consultar_cardapio
// ============================================

func consultarCardapioTool() mcp.Tool {
	return mcp.NewTool("consultar_cardapio",
		mcp.WithDescription("Consulta o cardápio de um restaurante"),
		mcp.WithString("tenant_id",
			mcp.Required(),
			mcp.Description("ID do restaurante"),
		),
		mcp.WithString("categoria",
			mcp.Description("Filtrar por categoria (ex: Lanches, Bebidas, Sobremesas)"),
		),
		mcp.WithBoolean("apenas_disponiveis",
			mcp.Description("Mostrar apenas itens disponíveis"),
			mcp.DefaultBool(true),
		),
	)
}

func (s *MCPServer) consultarCardapioHandler() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := getArguments(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tenantID, err := getStringRequired(args, "tenant_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		categoria, _ := getString(args, "categoria")
		apenasDisponiveis := true
		if val, ok := args["apenas_disponiveis"].(bool); ok {
			apenasDisponiveis = val
		}

		// Busca cardápio
		cardapio, err := s.getCardapio(tenantID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Erro ao buscar cardápio: %v", err)), nil
		}

		// Filtra
		var filtrados []service.ProdutoItem
		for _, item := range cardapio {
			if apenasDisponiveis && !item.Disponivel {
				continue
			}
			if categoria != "" && item.Categoria != categoria {
				continue
			}
			filtrados = append(filtrados, item)
		}

		if len(filtrados) == 0 {
			return mcp.NewToolResultText("Nenhum item encontrado no cardápio"), nil
		}

		// Formata resposta
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📋 Cardápio (%d itens):\n\n", len(filtrados)))

		for _, item := range filtrados {
			status := "✅"
			if !item.Disponivel {
				status = "❌"
			}
			sb.WriteString(fmt.Sprintf("%s **%s** - R$ %.2f\n", status, item.Nome, item.Preco))
			if item.Descricao != "" {
				sb.WriteString(fmt.Sprintf("   %s\n", item.Descricao))
			}
			sb.WriteString("\n")
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
}

// ============================================
// FUNÇÕES AUXILIARES
// ============================================

// getArguments converte os argumentos da request
func getArguments(request mcp.CallToolRequest) (map[string]interface{}, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("argumentos inválidos")
	}
	return args, nil
}

// getString extrai uma string dos argumentos
func getString(args map[string]interface{}, key string) (string, bool) {
	val, ok := args[key].(string)
	return val, ok
}

// getStringRequired extrai uma string obrigatória
func getStringRequired(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key].(string)
	if !ok || val == "" {
		return "", fmt.Errorf("'%s' é obrigatório", key)
	}
	return val, nil
}

// getItems extrai e valida a lista de itens
func getItems(args map[string]interface{}) ([]service.ItemPedidoInput, error) {
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

// formatarRespostaPedido monta a mensagem de confirmação
func (s *MCPServer) formatarRespostaPedido(pedido *service.PedidoConfirmado) string {
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
