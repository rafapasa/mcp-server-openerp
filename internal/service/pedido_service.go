// internal/service/pedido_service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
)

// PedidoService gerencia as operações de pedidos
type PedidoService struct {
	pedidoRepo      repository.PedidoRepository
	cardapioService CardapioServiceInterface
}

func NewPedidoService(
	pedidoRepo repository.PedidoRepository,
	cardapioService CardapioServiceInterface,
) PedidoServiceInterface {
	return &PedidoService{
		pedidoRepo:      pedidoRepo,
		cardapioService: cardapioService,
	}
}

// ItemPedidoInput representa um item do pedido (entrada)
type ItemPedidoInput struct {
	Nome          string  `json:"nome"`
	Quantidade    int     `json:"quantidade"`
	Observacao    string  `json:"observacao"`
	PrecoUnitario float64 `json:"preco_unitario,omitempty"`
}

// PedidoExtraido representa o pedido extraído pela IA
type PedidoExtraido struct {
	Itens       []ItemPedidoInput `json:"itens"`
	Bebidas     []ItemPedidoInput `json:"bebidas,omitempty"`
	Observacoes string            `json:"observacoes,omitempty"`
}

// PedidoConfirmado representa o pedido após processamento
type PedidoConfirmado struct {
	ID            int               `json:"id"`
	TenantID      string            `json:"tenant_id"`
	ClienteID     string            `json:"cliente_id"`
	ClienteNome   string            `json:"cliente_nome"`
	Itens         []ItemPedidoInput `json:"itens"`
	Total         float64           `json:"total"`
	TempoEstimado int               `json:"tempo_estimado"`
	Status        string            `json:"status"`
	CriadoEm      string            `json:"criado_em"`
}

// ProcessarPedido processa e salva um pedido
func (s *PedidoService) ProcessarPedido(
	ctx context.Context,
	tenantID, clienteID, clienteNome string,
	pedidoExtraido *PedidoExtraido,
) (*PedidoConfirmado, error) {

	// 1. Busca cardápio para validar e calcular preços
	cardapio, err := s.cardapioService.GetCardapio(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. Combina itens e bebidas
	todosItens := append(pedidoExtraido.Itens, pedidoExtraido.Bebidas...)

	// 3. Valida e calcula preços
	var itensComPreco []ItemPedidoInput
	total := 0.0

	for _, item := range todosItens {
		// Busca no cardápio
		existe, preco := s.cardapioService.ItemExisteNoCardapio(cardapio, item.Nome)
		if !existe {
			// Tenta encontrar similar
			similar := s.cardapioService.EncontrarItemSimilar(cardapio, item.Nome)
			if similar != "" {
				_, preco = s.cardapioService.ItemExisteNoCardapio(cardapio, similar)
				item.Nome = similar
				log.Printf("[Pedido] Item '%s' corrigido para '%s'", item.Nome, similar)
			} else {
				return nil, fmt.Errorf("item '%s' não encontrado no cardápio", item.Nome)
			}
		}

		item.PrecoUnitario = preco
		total += preco * float64(item.Quantidade)
		itensComPreco = append(itensComPreco, item)
	}

	// 4. Converte tenantID para uint
	var tenantIDUint uint
	if _, err := fmt.Sscan(tenantID, &tenantIDUint); err != nil {
		return nil, fmt.Errorf("tenant_id inválido: %w", err)
	}

	// 5. Prepara pedido para salvar
	itensJSON, _ := json.Marshal(itensComPreco)

	pedido := &models.Pedido{
		TenantID:        tenantIDUint,
		ClienteID:       clienteID,
		ClienteNome:     clienteNome,
		ClienteTelefone: clienteID, // usando o mesmo ID como telefone
		Itens:           itensJSON,
		Total:           total,
		Status:          models.StatusConfirmado,
		Observacoes:     pedidoExtraido.Observacoes,
		TempoEstimado:   s.CalcularTempoEstimado(itensComPreco),
		Origem:          models.OrigemWhatsApp,
	}

	// 6. Salva no banco
	if err := s.pedidoRepo.Create(ctx, pedido); err != nil {
		return nil, fmt.Errorf("erro ao salvar pedido: %w", err)
	}

	// 7. Monta resposta
	pedidoConfirmado := &PedidoConfirmado{
		ID:            int(pedido.ID),
		TenantID:      tenantID,
		ClienteID:     clienteID,
		ClienteNome:   clienteNome,
		Itens:         itensComPreco,
		Total:         total,
		TempoEstimado: pedido.TempoEstimado,
		Status:        pedido.Status,
		CriadoEm:      pedido.CreatedAt.Format("02/01/2006 15:04:05"),
	}

	log.Printf("[Pedido] Pedido #%d criado para tenant %s, total R$ %.2f",
		pedido.ID, tenantID, total)

	return pedidoConfirmado, nil
}

// calcularTempoEstimado calcula o tempo estimado baseado nos itens
func (s *PedidoService) CalcularTempoEstimado(itens []ItemPedidoInput) int {
	tempoBase := 15 // minutos
	tempoPorItem := 5

	totalItems := 0
	for _, item := range itens {
		totalItems += item.Quantidade
	}

	return tempoBase + (totalItems * tempoPorItem)
}

// AtualizarStatus atualiza o status de um pedido
func (s *PedidoService) AtualizarStatus(ctx context.Context, pedidoID uint, status string) error {
	return s.pedidoRepo.UpdateStatus(ctx, pedidoID, status)
}

// BuscarPedidosDoDia busca pedidos do dia para um tenant
func (s *PedidoService) BuscarPedidosDoDia(ctx context.Context, tenantID string) ([]models.Pedido, error) {
	var tenantIDUint uint
	if _, err := fmt.Sscan(tenantID, &tenantIDUint); err != nil {
		return nil, fmt.Errorf("tenant_id inválido: %w", err)
	}

	hoje := time.Now()
	inicio := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, hoje.Location())
	fim := inicio.Add(24 * time.Hour)

	return s.pedidoRepo.FindByTenantPeriodo(ctx, tenantIDUint, inicio, fim)
}
