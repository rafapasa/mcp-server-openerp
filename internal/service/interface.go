// internal/service/interfaces.go
package service

import (
	"context"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
)

// CarrinhoServiceInterface define as operações do serviço de carrinho
type CarrinhoServiceInterface interface {
	AdicionarItem(ctx context.Context, clienteID, tenantID uint, item dto.ItemCarrinho) error
	RemoverItem(ctx context.Context, clienteID, tenantID uint, itemCarrinho dto.ItemCarrinho, quantidade int) error
	GetCarrinho(ctx context.Context, clienteID, tenantID uint) (*dto.Carrinho, error)
	LimparCarrinho(ctx context.Context, clienteID, tenantID uint) error
	FinalizarCarrinho(ctx context.Context, clienteID, tenantID uint, clienteNome string) (*dto.PedidoConfirmado, error)
	CalcularTotal(carrinho *dto.Carrinho) float64
	CalcularTempoEstimado(carrinho *dto.Carrinho) int
	FormatResumoCarrinho(ctx context.Context, carrinho *dto.Carrinho) (string, error)
	FormatResumoCarrinhoByCliente(ctx context.Context, clienteID, tenantID uint) (string, error)
	FormatarPedidoConfirmado(pedido *dto.PedidoConfirmado) string
	ProcessarMensagem(ctx context.Context, clienteID, tenantID uint, input dto.MessageInput) (string, error)
}

// ============================================
// CARDAPIO SERVICE
// ============================================

type CardapioServiceInterface interface {
	// Existing methods
	GetCardapio(ctx context.Context, tenantID uint) ([]dto.ProdutoItem, error)
	BuscarProdutoPorNome(ctx context.Context, tenantID string, nome string) (*dto.ProdutoItem, error)
	BuscarProdutoPorIdNoCardapio(cardapio []dto.ProdutoItem, produtoID uint) (*dto.ProdutoItem, error)
	ItemExisteNoCardapio(cardapio []dto.ProdutoItem, nome string) (*dto.ProdutoItem, error)
	EncontrarItemSimilar(cardapio []dto.ProdutoItem, nome string) string
	FormatarCardapio(cardapio []dto.ProdutoItem) string

	// NEW: Dashboard methods
	ListWithFilters(ctx context.Context, tenantID uint, categoriaID *uint, disponivel *bool, nome string, page, limit int) ([]dto.ProdutoDTO, int64, error)
	FindByID(ctx context.Context, id uint) (*dto.ProdutoDTO, error)
}

// ============================================
// PEDIDO SERVICE
// ============================================

type PedidoServiceInterface interface {
	// Existing methods
	ProcessarPedido(ctx context.Context, tenantID, clienteID uint, clienteNome string, pedidoExtraido *dto.PedidoExtraido) (*dto.PedidoConfirmado, error)
	FindByTenant(ctx context.Context, tenantID uint) ([]dto.PedidoDTO, error)

	// NEW: Dashboard methods
	CountPedidosHoje(ctx context.Context, tenantID uint) (int64, error)
	CountPedidosSemana(ctx context.Context, tenantID uint) (int64, error)
	CountPorStatus(ctx context.Context, tenantID uint) (map[string]int64, error)
	CountPendentes(ctx context.Context, tenantID uint) (int64, error)
	FaturamentoHoje(ctx context.Context, tenantID uint) (float64, error)
	FaturamentoMes(ctx context.Context, tenantID uint) (float64, error)

	// NEW: List methods
	ListWithFilters(ctx context.Context, tenantID uint, clienteID uint, status string, dataInicio, dataFim time.Time, page, limit int) ([]dto.PedidoDTO, int64, error)
	FindByID(ctx context.Context, id uint) (*dto.PedidoDTO, error)
	ListByCliente(ctx context.Context, clienteID uint, page, limit int) ([]dto.PedidoDTO, int64, error)
	AtualizarStatusPedido(ctx context.Context, id uint, status string) (*dto.PedidoDTO, error)
	Create(ctx context.Context, req *dto.CriarPedidoRequest) (*dto.PedidoDTO, error)
}

// ============================================
// CLIENTE SERVICE
// ============================================

// ClienteServiceInterface define as operações do serviço de clientes
type ClienteServiceInterface interface {
	// CRUD Básico
	Create(ctx context.Context, req *dto.CriarClienteRequest) (*dto.ClienteDTO, error)
	FindByID(ctx context.Context, id uint) (*dto.ClienteDTO, error)
	FindByTelefone(ctx context.Context, telefone string, tenantID uint) (*dto.ClienteDTO, error)
	FindByTenant(ctx context.Context, tenantID uint) ([]dto.ClienteDTO, error)
	Update(ctx context.Context, id uint, req *dto.AtualizarClienteRequest) (*dto.ClienteDTO, error)
	Delete(ctx context.Context, id uint) error

	// Buscas Específicas
	BuscarOuCriarPorTelefone(ctx context.Context, tenantID uint, telefone, nomePerfil string) (*dto.ClienteDTO, error)
	BuscarPorNome(ctx context.Context, tenantID uint, nome string) ([]dto.ClienteDTO, error)
	BuscarPorStatus(ctx context.Context, tenantID uint, status string) ([]dto.ClienteDTO, error)
	BuscarInativos(ctx context.Context, tenantID uint, diasInatividade int) ([]dto.ClienteDTO, error)

	// Validação e Gestão de Clientes
	ValidarCliente(ctx context.Context, clienteID uint) (*dto.ClienteDTO, error)
	AtualizarUltimoPedido(ctx context.Context, clienteID uint) error
	AtualizarStatus(ctx context.Context, clienteID uint, status, motivo string) error
	InativarCliente(ctx context.Context, clienteID uint, motivo string) error
	ReativarCliente(ctx context.Context, clienteID uint) error

	// Endereços
	AdicionarEndereco(ctx context.Context, clienteID uint, req *dto.CriarEnderecoRequest) (*dto.EnderecoDTO, error)
	ListarEnderecos(ctx context.Context, clienteID uint) ([]dto.EnderecoDTO, error)
	DefinirEnderecoPrincipal(ctx context.Context, clienteID, enderecoID uint) error
	RemoverEndereco(ctx context.Context, clienteID, enderecoID uint) error

	// Documentos
	AtualizarDocumento(ctx context.Context, clienteID uint, inscricaoFederal string) error
	ValidarDocumento(inscricaoFederal string) (string, error)

	// Status
	IsAtivo(ctx context.Context, clienteID uint) (bool, error)
	GetStatus(ctx context.Context, clienteID uint) (string, error)

	// Utilitários
	ConverterParaDTO(cliente *models.Cliente) *dto.ClienteDTO
	ListWithFilters(ctx context.Context, tenantID uint, nome, telefone string, page, limit int) ([]dto.ClienteDTO, int64, error)
	CountByTenant(ctx context.Context, tenantID uint) (int64, error)
}

type AuthServiceInterface interface {
	Authenticate(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	ValidateToken(tokenString string) (*Claims, error)
}

type TenantServiceInterface interface {
	GetByID(ctx context.Context, id uint) (*dto.TenantDTO, error)
	GetByCNPJ(ctx context.Context, cnpj string) (*dto.TenantDTO, error)
	List(ctx context.Context) ([]dto.TenantDTO, error)
	Create(ctx context.Context, input dto.CreateTenantDTO) (*dto.TenantDTO, error)
	GetPromptContext(ctx context.Context, tenantID uint) (nome, segmento string, err error)
}
