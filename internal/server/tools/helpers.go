// internal/server/tools/helpers.go - COMPLETO - SEGURO ID MySQL
package tools

import (
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
)

func GetArguments(request mcp.CallToolRequest) (map[string]interface{}, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("arguments inválidos")
	}
	return args, nil
}

func GetString(args map[string]interface{}, key string) (string, bool) {
	val, ok := args[key].(string)
	return val, ok
}

func GetStringRequired(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key].(string)
	if !ok || val == "" {
		return "", fmt.Errorf("'%s' é obrigatório", key)
	}
	return val, nil
}

// GetUintRequired agora aceita string OU number (float64/int) - MCP manda number como float64
func GetUintRequired(args map[string]interface{}, key string) (uint, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return 0, fmt.Errorf("'%s' é obrigatório", key)
	}
	switch v := raw.(type) {
	case string:
		if v == "" {
			return 0, fmt.Errorf("'%s' é obrigatório", key)
		}
		parsed, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("'%s' deve ser número válido", key)
		}
		return uint(parsed), nil
	case float64:
		if v <= 0 {
			return 0, fmt.Errorf("'%s' deve ser maior que zero", key)
		}
		return uint(v), nil
	case float32:
		return uint(v), nil
	case int:
		if v <= 0 {
			return 0, fmt.Errorf("'%s' deve ser maior que zero", key)
		}
		return uint(v), nil
	case int64:
		return uint(v), nil
	case uint:
		return v, nil
	case uint64:
		return uint(v), nil
	default:
		return 0, fmt.Errorf("'%s' deve ser número válido, recebido %T", key, raw)
	}
}

func GetIntRequired(args map[string]interface{}, key string) (int, error) {
	u, err := GetUintRequired(args, key)
	return int(u), err
}

// GetItems LEGADO - mantido pra não quebrar, mas NÃO usar pro fluxo novo
// Fluxo novo valida ID MySQL via BuscarProdutoPorIdNoCardapio no handler, não aqui
func GetItems(args map[string]interface{}) ([]dto.ItemPedidoInput, error) {
	itensRaw, ok := args["itens"].([]interface{})
	if !ok || len(itensRaw) == 0 {
		return nil, fmt.Errorf("itens é obrigatório")
	}

	var itens []dto.ItemPedidoInput
	for _, itemRaw := range itensRaw {
		itemMap, ok := itemRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Suporta tanto fluxo novo (produto_id) quanto legado (nome)
		var produtoItem dto.ProdutoItem

		if pid, ok := itemMap["produto_id"]; ok {
			var id uint
			switch v := pid.(type) {
			case float64:
				id = uint(v)
			case string:
				parsed, _ := strconv.ParseUint(v, 10, 32)
				id = uint(parsed)
			case int:
				id = uint(v)
			}
			if id > 0 {
				produtoItem.ID = id
			}
		}

		// Legado - se ainda vier nome, mantém pra log mas não valida por nome
		if produtoItem.ID == 0 {
			if nome, ok := itemMap["nome"].(string); ok && nome != "" {
				produtoItem.Nome = nome
			}
		}

		qtd := 1
		if q, ok := itemMap["quantidade"].(float64); ok && q > 0 {
			qtd = int(q)
		}
		obs, _ := itemMap["observacao"].(string)

		if produtoItem.ID != 0 || produtoItem.Nome != "" {
			itens = append(itens, dto.ItemPedidoInput{
				ProdutoItem: produtoItem,
				Quantidade:  qtd,
				Observacao:  obs,
			})
		}
	}

	if len(itens) == 0 {
		return nil, fmt.Errorf("nenhum item válido encontrado")
	}
	return itens, nil
}
