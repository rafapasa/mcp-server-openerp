// internal/service/pedido_service.go - CORRIGIDO
// Regra: só chama seu próprio repo (pedidoRepo) + cardapioService
// Não chama clienteRepo nem enderecoRepo
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
	pedidoRepo         repository.PedidoRepository
	pagamentoRepo      repository.PedidoPagamentoRepository
	formaPagamentoRepo repository.FormaPagamentoRepository
	cardapioService    CardapioServiceInterface
}

func NewPedidoService(
	pedidoRepo repository.PedidoRepository,
	pagamentoRepo repository.PedidoPagamentoRepository,
	formaPagamentoRepo repository.FormaPagamentoRepository,
	cardapioService CardapioServiceInterface,
) PedidoServiceInterface {
	return &PedidoService{
		pedidoRepo:         pedidoRepo,
		pagamentoRepo:      pagamentoRepo,
		formaPagamentoRepo: formaPagamentoRepo,
		cardapioService:    cardapioService,
	}
}

// ProcessarPedido - assinatura original mantida para compatibilidade
func (s *PedidoService) ProcessarPedido(
	ctx context.Context,
	tenantID, clienteID uint,
	clienteNome string,
	pedidoExtraido *dto.PedidoExtraido,
) (*dto.PedidoConfirmado, error) {
	return s.ProcessarPedidoComEndereco(ctx, tenantID, clienteID, clienteNome, pedidoExtraido, nil)
}

// ProcessarPedidoComEndereco - NOVO, sem chamar outros repos
func (s *PedidoService) ProcessarPedidoComEndereco(
	ctx context.Context,
	tenantID, clienteID uint,
	clienteNome string,
	pedidoExtraido *dto.PedidoExtraido,
	enderecoEntregaID *uint,
) (*dto.PedidoConfirmado, error) {
	return s.ProcessarPedidoComEnderecoEPagamentos(ctx, tenantID, clienteID, clienteNome, pedidoExtraido, enderecoEntregaID, nil)
}

func (s *PedidoService) ProcessarPedidoComEnderecoEPagamentos(
	ctx context.Context,
	tenantID, clienteID uint,
	clienteNome string,
	pedidoExtraido *dto.PedidoExtraido,
	enderecoEntregaID *uint,
	pagamentos []dto.PedidoPagamentoInput,
) (*dto.PedidoConfirmado, error) {
	logger.Info(
		ctx, "Processando pedido",
		zap.Uint("tenant_id", tenantID),
		zap.Uint("cliente_id", clienteID),
		zap.String("cliente_nome", clienteNome),
		zap.Int("itens_count", len(pedidoExtraido.Itens)),
		zap.Any("endereco_id", enderecoEntregaID),
	)

	cardapio, err := s.cardapioService.GetCardapio(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, "Erro ao buscar cardápio", zap.Error(err))
		return nil, err
	}

	// Junta itens normais + bebidas se existirem no DTO
	todosItens := append([]dto.ItemPedidoInput{}, pedidoExtraido.Itens...)
	todosItens = append(todosItens, pedidoExtraido.Bebidas...)

	var itensComPreco []dto.ItemPedidoInput
	total := 0.0

	for _, item := range todosItens {
		nomeBusca := item.ProdutoItem.Nome
		if nomeBusca == "" {
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

		preco := prodCardapio.Preco
		// se já veio com preço do carrinho, respeita (promoção já calculada)
		if item.PrecoUnitario != 0 {
			preco = item.PrecoUnitario
		}

		item.PrecoUnitario = preco
		item.ProdutoItem.Nome = prodCardapio.Nome
		item.ProdutoItem.ID = prodCardapio.ID
		total += preco * float64(item.Quantidade)
		itensComPreco = append(itensComPreco, item)
	}

	if len(itensComPreco) == 0 {
		return nil, fmt.Errorf("nenhum item válido para pedido")
	}
	pagamentosModel, err := s.prepararPagamentos(ctx, tenantID, total, pagamentos)
	if err != nil {
		return nil, err
	}

	itensJSON, err := json.Marshal(itensComPreco)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar itens: %w", err)
	}

	pedido := &models.Pedido{
		TenantID:          tenantID,
		ClienteID:         &clienteID,
		EnderecoEntregaID: enderecoEntregaID, // NOVO - pode ser nil (retirada) ou ID válido
		ClienteNome:       clienteNome,
		Itens:             itensJSON,
		Total:             total,
		Status:            models.StatusConfirmado,
		Observacoes:       pedidoExtraido.Observacoes,
		TempoEstimado:     s.CalcularTempoEstimado(itensComPreco),
		Origem:            models.OrigemWhatsApp,
	}

	if err := s.pedidoRepo.Create(ctx, pedido); err != nil {
		logger.Error(ctx, "Erro ao salvar pedido", zap.Error(err))
		return nil, fmt.Errorf("erro ao salvar pedido: %w", err)
	}
	for i := range pagamentosModel {
		pagamentosModel[i].PedidoID = pedido.ID
	}
	if len(pagamentosModel) > 0 {
		if err := s.pagamentoRepo.CreateMany(ctx, pagamentosModel); err != nil {
			return nil, fmt.Errorf("erro ao registrar pagamentos: %w", err)
		}
	}

	// Não busca cliente nem endereço aqui - respeita separação de repos
	// Quem precisar do endereço completo, busca via ClienteService no carrinho_service
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
		// EnderecoEntrega será preenchido pelo CarrinhoService via ClienteService se necessário
	}

	logger.Info(ctx, "Pedido criado com sucesso", zap.Uint("pedido_id", pedido.ID), zap.Float64("total", total), zap.Any("endereco_id", enderecoEntregaID))
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
	if status == models.StatusEntregue && s.pagamentoRepo != nil {
		if err := s.pagamentoRepo.MarcarPendentesComoPagos(ctx, id); err != nil {
			return nil, fmt.Errorf("erro ao atualizar pagamentos do pedido: %w", err)
		}
	}
	result := s.converterParaDTO(pedido)
	result.Pagamentos = s.pagamentosDTO(ctx, id)
	return &result, nil
}

func (s *PedidoService) Create(ctx context.Context, req *dto.CriarPedidoRequest) (*dto.PedidoDTO, error) {
	logger.Info(ctx, "Criando novo pedido via API", zap.Uint("tenant_id", req.TenantID), zap.Int("itens_count", len(req.Itens)))
	itensJSON, _ := json.Marshal(req.Itens)
	pedido := &models.Pedido{
		TenantID:          req.TenantID,
		ClienteID:         req.ClienteID,
		EnderecoEntregaID: req.EnderecoEntregaID,
		Itens:             itensJSON,
		Total:             req.TotalFloat64(),
		Status:            models.StatusPendente,
		Origem:            models.OrigemAPI,
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
			ID:          pedido.EnderecoEntrega.ID,
			ClienteID:   pedido.EnderecoEntrega.ClienteID,
			CEP:         pedido.EnderecoEntrega.CEP,
			Logradouro:  pedido.EnderecoEntrega.Logradouro,
			Numero:      pedido.EnderecoEntrega.Numero,
			Complemento: pedido.EnderecoEntrega.Complemento,
			Bairro:      pedido.EnderecoEntrega.Bairro,
			Cidade:      pedido.EnderecoEntrega.Cidade,
			Estado:      pedido.EnderecoEntrega.Estado,
			Pais:        pedido.EnderecoEntrega.Pais,
			Referencia:  pedido.EnderecoEntrega.Referencia,
			Latitude:    pedido.EnderecoEntrega.Latitude,
			Longitude:   pedido.EnderecoEntrega.Longitude,
			Tipo:        pedido.EnderecoEntrega.Tipo,
			Principal:   pedido.EnderecoEntrega.Principal,
			Observacoes: pedido.EnderecoEntrega.Observacoes,
			CreatedAt:   pedido.EnderecoEntrega.CreatedAt,
			UpdatedAt:   pedido.EnderecoEntrega.UpdatedAt,
		}
	}
	return dto.PedidoDTO{
		ID:                pedido.ID,
		TenantID:          pedido.TenantID,
		ClienteID:         pedido.ClienteID,
		ClienteNome:       pedido.ClienteNome,
		ClienteTelefone:   pedido.ClienteTelefone,
		EnderecoEntregaID: pedido.EnderecoEntregaID,
		EnderecoEntrega:   endereco,
		Itens:             itens,
		Total:             pedido.Total,
		Status:            pedido.Status,
		Observacoes:       pedido.Observacoes,
		TempoEstimado:     pedido.TempoEstimado,
		Origem:            pedido.Origem,
		CreatedAt:         pedido.CreatedAt,
		UpdatedAt:         pedido.UpdatedAt,
		Pagamentos:        pagamentosModelDTO(pedido.Pagamentos),
	}
}

func (s *PedidoService) prepararPagamentos(ctx context.Context, tenantID uint, total float64, inputs []dto.PedidoPagamentoInput) ([]models.PedidoPagamento, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	if s.pagamentoRepo == nil || s.formaPagamentoRepo == nil {
		return nil, fmt.Errorf("repositórios de pagamento não configurados")
	}
	pagamentos := make([]models.PedidoPagamento, 0, len(inputs))
	totalPagamentos := 0.0
	for _, input := range inputs {
		if input.Valor <= 0 {
			return nil, fmt.Errorf("valor do pagamento deve ser maior que zero")
		}
		forma, err := s.formaPagamentoRepo.FindByID(ctx, input.FormaPagamentoID)
		if err != nil {
			return nil, fmt.Errorf("forma de pagamento não encontrada: %w", err)
		}
		if forma.TenantID != tenantID || !forma.Ativo {
			return nil, fmt.Errorf("forma de pagamento inválida para o tenant")
		}
		if forma.Tipo != models.TipoPagamentoDinheiro && input.TrocoPara != nil {
			return nil, fmt.Errorf("troco só pode ser informado para dinheiro")
		}
		if input.TrocoPara != nil && *input.TrocoPara < input.Valor {
			return nil, fmt.Errorf("troco para deve ser maior ou igual ao valor do pagamento")
		}
		totalPagamentos += input.Valor
		pagamentos = append(pagamentos, models.PedidoPagamento{
			FormaPagamentoID: input.FormaPagamentoID,
			Valor:            input.Valor, TrocoPara: input.TrocoPara,
			Status: models.StatusPagamentoPendente,
		})
	}
	if totalPagamentos < total-0.01 || totalPagamentos > total+0.01 {
		return nil, fmt.Errorf("a soma dos pagamentos deve corresponder ao total do pedido")
	}
	return pagamentos, nil
}

func (s *PedidoService) pagamentosDTO(ctx context.Context, pedidoID uint) []dto.PedidoPagamentoDTO {
	if s.pagamentoRepo == nil {
		return nil
	}
	pagamentos, err := s.pagamentoRepo.FindByPedido(ctx, pedidoID)
	if err != nil {
		return nil
	}
	return pagamentosModelDTO(pagamentos)
}

func pagamentosModelDTO(pagamentos []models.PedidoPagamento) []dto.PedidoPagamentoDTO {
	result := make([]dto.PedidoPagamentoDTO, len(pagamentos))
	for i := range pagamentos {
		item := &pagamentos[i]
		result[i] = dto.PedidoPagamentoDTO{
			ID: item.ID, PedidoID: item.PedidoID, FormaPagamentoID: item.FormaPagamentoID,
			Valor: item.Valor, TrocoPara: item.TrocoPara, Status: item.Status,
		}
		if item.FormaPagamento.ID != 0 {
			forma := formaPagamentoDTO(&item.FormaPagamento)
			result[i].FormaPagamento = &forma
		}
	}
	return result
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
