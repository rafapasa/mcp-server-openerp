// internal/service/pedido_service.go - CORRIGIDO MANTENDO ASSINATURAS DA INTERFACE
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"go.uber.org/zap"
)

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

func (s *PedidoService) ProcessarPedido(
	ctx context.Context,
	tenantID, clienteID uint,
	clienteNome string,
	pedidoExtraido *dto.PedidoExtraido,
) (*dto.PedidoConfirmado, error) {
	logger.Info(
		ctx, "Processando pedido",
		zap.Uint("tenant_id", tenantID),
		zap.Uint("cliente_id", clienteID),
		zap.String("cliente_nome", clienteNome),
		zap.Int("itens_count", len(pedidoExtraido.Itens)),
	)

	cardapio, err := s.cardapioService.GetCardapio(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, "Erro ao buscar cardápio", zap.Error(err))
		return nil, err
	}

	todosItens := append(pedidoExtraido.Itens, pedidoExtraido.Bebidas...)

	var itensComPreco []dto.ItemPedidoInput
	total := 0.0

	for _, item := range todosItens {
		// Nome pode estar em item.Nome ou item.ProdutoItem.Nome dependendo do DTO
		nomeBusca := item.ProdutoItem.Nome
		if nomeBusca == "" && item.ProdutoItem.Nome != "" {
			nomeBusca = item.ProdutoItem.Nome
		}

		prodCardapio, err := s.cardapioService.ItemExisteNoCardapio(cardapio, nomeBusca)
		if err != nil || prodCardapio == nil {
			similar := s.cardapioService.EncontrarItemSimilar(cardapio, nomeBusca)
			if similar != "" {
				if pSimilar, err := s.cardapioService.ItemExisteNoCardapio(cardapio, similar); err == nil && pSimilar != nil {
					prodCardapio = pSimilar
					logger.Info(ctx, "Item corrigido", zap.String("original", nomeBusca), zap.String("corrigido", similar))
				}
			}
		}

		if prodCardapio == nil {
			logger.Warn(ctx, "Item não encontrado no cardápio", zap.String("item", nomeBusca))
			return nil, fmt.Errorf("item '%s' não encontrado no cardápio", nomeBusca)
		}

		// Preço vem do cardápio validado do MySQL
		preco := prodCardapio.Preco
		if item.PrecoUnitario != 0 {
			preco = item.PrecoUnitario
		}

		item.PrecoUnitario = preco
		item.ProdutoItem.Nome = prodCardapio.Nome
		total += preco * float64(item.Quantidade)
		itensComPreco = append(itensComPreco, item)
	}

	itensJSON, _ := json.Marshal(itensComPreco)

	pedido := &models.Pedido{
		TenantID:        tenantID,
		ClienteID:       &clienteID,
		ClienteNome:     clienteNome,
		ClienteTelefone: fmt.Sprint(clienteID),
		Itens:           itensJSON,
		Total:           total,
		Status:          models.StatusConfirmado,
		Observacoes:     pedidoExtraido.Observacoes,
		TempoEstimado:   s.CalcularTempoEstimado(itensComPreco),
		Origem:          models.OrigemWhatsApp,
	}

	if err := s.pedidoRepo.Create(ctx, pedido); err != nil {
		logger.Error(ctx, "Erro ao salvar pedido", zap.Error(err))
		return nil, fmt.Errorf("erro ao salvar pedido: %w", err)
	}

	pedidoConfirmado := &dto.PedidoConfirmado{
		ID:            int(pedido.ID),
		TenantID:      fmt.Sprint(tenantID),
		ClienteID:     fmt.Sprint(clienteID),
		ClienteNome:   clienteNome,
		Itens:         itensComPreco,
		Total:         total,
		TempoEstimado: pedido.TempoEstimado,
		Status:        pedido.Status,
		CriadoEm:      pedido.CreatedAt.Format("02/01/2006 15:04:05"),
	}

	logger.Info(ctx, "Pedido criado com sucesso", zap.Uint("pedido_id", pedido.ID), zap.Float64("total", total))
	return pedidoConfirmado, nil
}

func (s *PedidoService) CalcularTempoEstimado(itens []dto.ItemPedidoInput) int {
	totalItems := 0
	for _, item := range itens {
		totalItems += item.Quantidade
	}
	return 15 + (totalItems * 5)
}

func (s *PedidoService) AtualizarStatus(ctx context.Context, pedidoID uint, status string) error {
	_, err := s.pedidoRepo.UpdateStatus(ctx, pedidoID, status)
	return err
}

func (s *PedidoService) BuscarPedidosDoDia(ctx context.Context, tenantID uint) ([]models.Pedido, error) {
	hoje := time.Now()
	inicio := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, hoje.Location())
	fim := inicio.Add(24 * time.Hour)
	return s.pedidoRepo.FindByTenantPeriodo(ctx, tenantID, inicio, fim)
}

func (s *PedidoService) CountPedidosHoje(ctx context.Context, tenantID uint) (int64, error) {
	hoje := time.Now()
	inicio := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, hoje.Location())
	fim := inicio.Add(24 * time.Hour)
	return s.pedidoRepo.CountByPeriodo(ctx, tenantID, inicio, fim)
}

func (s *PedidoService) CountPedidosSemana(ctx context.Context, tenantID uint) (int64, error) {
	hoje := time.Now()
	inicio := hoje.AddDate(0, 0, -7)
	return s.pedidoRepo.CountByPeriodo(ctx, tenantID, inicio, hoje)
}

func (s *PedidoService) CountPorStatus(ctx context.Context, tenantID uint) (map[string]int64, error) {
	return s.pedidoRepo.CountGroupByStatus(ctx, tenantID)
}

func (s *PedidoService) CountPendentes(ctx context.Context, tenantID uint) (int64, error) {
	return s.pedidoRepo.CountByStatus(ctx, tenantID, "pendente")
}

func (s *PedidoService) FaturamentoHoje(ctx context.Context, tenantID uint) (float64, error) {
	hoje := time.Now()
	inicio := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, hoje.Location())
	fim := inicio.Add(24 * time.Hour)
	return s.pedidoRepo.SumTotalByPeriodo(ctx, tenantID, inicio, fim)
}

func (s *PedidoService) FaturamentoMes(ctx context.Context, tenantID uint) (float64, error) {
	hoje := time.Now()
	inicio := time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, hoje.Location())
	fim := inicio.AddDate(0, 1, 0)
	return s.pedidoRepo.SumTotalByPeriodo(ctx, tenantID, inicio, fim)
}

func (s *PedidoService) ListWithFilters(ctx context.Context, tenantID uint, clienteID uint, status string, dataInicio, dataFim time.Time, page, limit int) ([]dto.PedidoDTO, int64, error) {
	offset := (page - 1) * limit
	pedidos, total, err := s.pedidoRepo.FindWithFilters(ctx, tenantID, clienteID, status, dataInicio, dataFim, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]dto.PedidoDTO, len(pedidos))
	for i, p := range pedidos {
		result[i] = s.converterParaDTO(&p)
	}
	return result, total, nil
}

func (s *PedidoService) FindByID(ctx context.Context, id uint) (*dto.PedidoDTO, error) {
	pedido, err := s.pedidoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result := s.converterParaDTO(pedido)
	return &result, nil
}

func (s *PedidoService) ListByCliente(ctx context.Context, clienteID uint, page, limit int) ([]dto.PedidoDTO, int64, error) {
	offset := (page - 1) * limit
	pedidos, total, err := s.pedidoRepo.FindByCliente(ctx, clienteID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]dto.PedidoDTO, len(pedidos))
	for i, p := range pedidos {
		result[i] = s.converterParaDTO(&p)
	}
	return result, total, nil
}

func (s *PedidoService) AtualizarStatusPedido(ctx context.Context, id uint, status string) (*dto.PedidoDTO, error) {
	pedido, err := s.pedidoRepo.UpdateStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}
	result := s.converterParaDTO(pedido)
	return &result, nil
}

func (s *PedidoService) Create(ctx context.Context, req *dto.CriarPedidoRequest) (*dto.PedidoDTO, error) {
	logger.Info(ctx, "Criando novo pedido via API", zap.Uint("tenant_id", req.TenantID), zap.Int("itens_count", len(req.Itens)))
	itensJSON, _ := json.Marshal(req.Itens)
	pedido := &models.Pedido{
		TenantID:  req.TenantID,
		ClienteID: req.ClienteID,
		Itens:     itensJSON,
		Total:     req.TotalFloat64(),
		Status:    models.StatusPendente,
		Origem:    models.OrigemAPI,
	}
	if err := s.pedidoRepo.Create(ctx, pedido); err != nil {
		return nil, err
	}
	result := s.converterParaDTO(pedido)
	return &result, nil
}

func (s *PedidoService) converterParaDTO(pedido *models.Pedido) dto.PedidoDTO {
	var itens []dto.ItemPedidoDTO
	if pedido.Itens != nil {
		_ = json.Unmarshal(pedido.Itens, &itens)
	}
	var endereco *dto.EnderecoDTO
	if pedido.EnderecoEntrega != nil {
		endereco = &dto.EnderecoDTO{
			ID: pedido.EnderecoEntrega.ID, ClienteID: pedido.EnderecoEntrega.ClienteID,
			CEP: pedido.EnderecoEntrega.CEP, Logradouro: pedido.EnderecoEntrega.Logradouro,
			Numero: pedido.EnderecoEntrega.Numero, Complemento: pedido.EnderecoEntrega.Complemento,
			Bairro: pedido.EnderecoEntrega.Bairro, Cidade: pedido.EnderecoEntrega.Cidade,
			Estado: pedido.EnderecoEntrega.Estado, Pais: pedido.EnderecoEntrega.Pais,
			Referencia: pedido.EnderecoEntrega.Referencia, Latitude: pedido.EnderecoEntrega.Latitude,
			Longitude: pedido.EnderecoEntrega.Longitude, Tipo: pedido.EnderecoEntrega.Tipo,
			Principal: pedido.EnderecoEntrega.Principal, Observacoes: pedido.EnderecoEntrega.Observacoes,
			CreatedAt: pedido.EnderecoEntrega.CreatedAt, UpdatedAt: pedido.EnderecoEntrega.UpdatedAt,
		}
	}
	return dto.PedidoDTO{
		ID: pedido.ID, TenantID: pedido.TenantID, ClienteID: pedido.ClienteID,
		ClienteNome: pedido.ClienteNome, ClienteTelefone: pedido.ClienteTelefone,
		EnderecoEntregaID: pedido.EnderecoEntregaID, EnderecoEntrega: endereco,
		Itens: itens, Total: pedido.Total, Status: pedido.Status,
		Observacoes: pedido.Observacoes, TempoEstimado: pedido.TempoEstimado,
		Origem: pedido.Origem, CreatedAt: pedido.CreatedAt, UpdatedAt: pedido.UpdatedAt,
	}
}

func (s *PedidoService) FindByTenant(ctx context.Context, tenantID uint) ([]dto.PedidoDTO, error) {
	if tenantID == 0 {
		return nil, fmt.Errorf("tenant_id não informado")
	}
	pedidos, _, err := s.pedidoRepo.FindByTenant(ctx, tenantID, 100, 0)
	if err != nil {
		return nil, err
	}
	result := make([]dto.PedidoDTO, len(pedidos))
	for i, p := range pedidos {
		result[i] = s.converterParaDTO(&p)
	}
	return result, nil
}
