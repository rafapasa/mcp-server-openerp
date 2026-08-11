// internal/service/carrinho_service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	TTLCarrinho = 3600 // 1 hora em segundos
)

// ItemCarrinho representa um item no carrinho
type ItemCarrinho struct {
	Nome       string  `json:"nome"`
	Quantidade int     `json:"quantidade"`
	Observacao string  `json:"observacao,omitempty"`
	Preco      float64 `json:"preco"`
}

// Carrinho representa o carrinho de um cliente
type Carrinho struct {
	ClienteID string         `json:"cliente_id"`
	TenantID  string         `json:"tenant_id"`
	Itens     []ItemCarrinho `json:"itens"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// CarrinhoService gerencia operações do carrinho no Redis
type CarrinhoService struct {
	cache           *redis.Client
	cardapioService CardapioServiceInterface
	pedidoService   PedidoServiceInterface
}

// NewCarrinhoService cria um novo service de carrinho
func NewCarrinhoService(cache *redis.Client, cardapioService CardapioServiceInterface, pedidoService PedidoServiceInterface) CarrinhoServiceInterface {
	return &CarrinhoService{
		cache:           cache,
		cardapioService: cardapioService,
		pedidoService:   pedidoService,
	}
}

// getKey retorna a chave do Redis para o carrinho
func (s *CarrinhoService) getKey(clienteID, tenantID string) string {
	return fmt.Sprintf("carrinho:%s:%s", tenantID, clienteID)
}

// GetCarrinho busca o carrinho do cliente
func (s *CarrinhoService) GetCarrinho(clienteID, tenantID string) (*Carrinho, error) {
	key := s.getKey(clienteID, tenantID)

	data, err := s.cache.Get(context.Background(), key).Result()
	if err == redis.Nil {
		// Carrinho vazio
		return &Carrinho{
			ClienteID: clienteID,
			TenantID:  tenantID,
			Itens:     []ItemCarrinho{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar carrinho: %w", err)
	}

	var carrinho Carrinho
	if err := json.Unmarshal([]byte(data), &carrinho); err != nil {
		return nil, fmt.Errorf("erro ao desserializar carrinho: %w", err)
	}

	return &carrinho, nil
}

// saveCarrinho salva o carrinho no Redis
func (s *CarrinhoService) saveCarrinho(carrinho *Carrinho) error {
	carrinho.UpdatedAt = time.Now()

	data, err := json.Marshal(carrinho)
	if err != nil {
		return fmt.Errorf("erro ao serializar carrinho: %w", err)
	}

	key := s.getKey(carrinho.ClienteID, carrinho.TenantID)

	if err := s.cache.Set(context.Background(), key, data, TTLCarrinho*time.Second).Err(); err != nil {
		return fmt.Errorf("erro ao salvar carrinho: %w", err)
	}

	return nil
}

// AdicionarItem adiciona um item ao carrinho
func (s *CarrinhoService) AdicionarItem(clienteID, tenantID string, item ItemCarrinho) error {
	carrinho, err := s.GetCarrinho(clienteID, tenantID)
	if err != nil {
		return err
	}

	// Verifica se o item já existe no carrinho
	for i, existingItem := range carrinho.Itens {
		if existingItem.Nome == item.Nome {
			carrinho.Itens[i].Quantidade += item.Quantidade
			// Mantém a observação mais recente
			if item.Observacao != "" {
				carrinho.Itens[i].Observacao = item.Observacao
			}
			return s.saveCarrinho(carrinho)
		}
	}

	// Adiciona novo item
	carrinho.Itens = append(carrinho.Itens, item)
	return s.saveCarrinho(carrinho)
}

// RemoverItem remove um item do carrinho
func (s *CarrinhoService) RemoverItem(clienteID, tenantID string, nome string, quantidade int) error {
	carrinho, err := s.GetCarrinho(clienteID, tenantID)
	if err != nil {
		return err
	}

	for i, item := range carrinho.Itens {
		if item.Nome == nome {
			if quantidade == 0 || quantidade >= item.Quantidade {
				// Remove o item completamente
				carrinho.Itens = append(carrinho.Itens[:i], carrinho.Itens[i+1:]...)
			} else {
				// Reduz a quantidade
				carrinho.Itens[i].Quantidade -= quantidade
			}
			return s.saveCarrinho(carrinho)
		}
	}

	return fmt.Errorf("item '%s' não encontrado no carrinho", nome)
}

// LimparCarrinho limpa todo o carrinho
func (s *CarrinhoService) LimparCarrinho(clienteID, tenantID string) error {
	key := s.getKey(clienteID, tenantID)
	return s.cache.Del(context.Background(), key).Err()
}

// CalcularTotal calcula o total do carrinho
func (s *CarrinhoService) CalcularTotal(carrinho *Carrinho) float64 {
	total := 0.0
	for _, item := range carrinho.Itens {
		total += item.Preco * float64(item.Quantidade)
	}
	return total
}

// CalcularTempoEstimado calcula o tempo estimado do carrinho
func (s *CarrinhoService) CalcularTempoEstimado(carrinho *Carrinho) int {
	if len(carrinho.Itens) == 0 {
		return 0
	}

	tempoBase := 15 // minutos
	tempoPorItem := 5

	totalItems := 0
	for _, item := range carrinho.Itens {
		totalItems += item.Quantidade
	}

	return tempoBase + (totalItems * tempoPorItem)
}

// FinalizarCarrinho finaliza o pedido
func (s *CarrinhoService) FinalizarCarrinho(ctx context.Context, clienteID, tenantID, clienteNome string) (*PedidoConfirmado, error) {
	carrinho, err := s.GetCarrinho(clienteID, tenantID)
	if err != nil {
		return nil, err
	}

	if len(carrinho.Itens) == 0 {
		return nil, fmt.Errorf("carrinho vazio")
	}

	// Converte itens do carrinho para PedidoExtraido
	pedidoExtraido := &PedidoExtraido{}
	for _, item := range carrinho.Itens {
		pedidoExtraido.Itens = append(pedidoExtraido.Itens, ItemPedidoInput{
			Nome:          item.Nome,
			Quantidade:    item.Quantidade,
			Observacao:    item.Observacao,
			PrecoUnitario: item.Preco,
		})
	}

	// Processa o pedido
	pedidoConfirmado, err := s.pedidoService.ProcessarPedido(ctx, tenantID, clienteID, clienteNome, pedidoExtraido)
	if err != nil {
		return nil, err
	}

	// Limpa o carrinho
	if err := s.LimparCarrinho(clienteID, tenantID); err != nil {
		// Log do erro mas não falha o pedido
		log.Printf("[Carrinho] Erro ao limpar carrinho após finalizar: %v", err)
	}

	return pedidoConfirmado, nil
}
