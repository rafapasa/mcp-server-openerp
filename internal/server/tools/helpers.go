// internal/server/tools/helpers.go
package tools

import (
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
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

// GetUintRequired extrai um uint obrigatório
func GetUintRequired(args map[string]interface{}, key string) (uint, error) {
	val, ok := args[key].(string)
	if !ok || val == "" {
		return 0, fmt.Errorf("'%s' é obrigatório", key)
	}
	uintVal, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("'%s' deve ser um número válido", key)
	}
	return uint(uintVal), nil
}

// GetIntRequired extrai um int obrigatório
func GetIntRequired(args map[string]interface{}, key string) (int, error) {
	val, ok := args[key].(string)
	if !ok || val == "" {
		return 0, fmt.Errorf("'%s' é obrigatório", key)
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("'%s' deve ser um número válido", key)
	}
	return intVal, nil
}

// GetItems extrai e valida a lista de itens
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

		nome, _ := itemMap["nome"].(string)
		qtd, _ := itemMap["quantidade"].(float64)
		obs, _ := itemMap["observacao"].(string)

		if nome != "" && qtd > 0 {
			itens = append(itens, dto.ItemPedidoInput{
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
