package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/helpers"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"github.com/rafapasa/mcp-server-openerp/internal/repository"
	"go.uber.org/zap"
)

const (
	TTLCarrinho           = 3600
	limiteCardapioPequeno = 80
)

type CarrinhoService struct {
	cache           database.RedisInterface
	cardapioService CardapioServiceInterface
	pedidoService   PedidoServiceInterface
	clienteService  ClienteServiceInterface
	produtoRepo     repository.ProdutoRepository
	llmService      LLMServiceInterface
}

func NewCarrinhoService(
	cache database.RedisInterface,
	cardapioService CardapioServiceInterface,
	pedidoService PedidoServiceInterface,
	produtoRepo repository.ProdutoRepository,
	llmService LLMServiceInterface,
	clienteService ClienteServiceInterface,
) CarrinhoServiceInterface {
	return &CarrinhoService{
		cache:           cache,
		cardapioService: cardapioService,
		pedidoService:   pedidoService,
		clienteService:  clienteService,
		produtoRepo:     produtoRepo,
		llmService:      llmService,
	}
}

func (s *CarrinhoService) getKey(clienteID, tenantID uint) string {
	return fmt.Sprintf("carrinho:%d:%d", tenantID, clienteID)
}

func (s *CarrinhoService) GetCarrinho(ctx context.Context, clienteID, tenantID uint) (*dto.Carrinho, error) {
	key := s.getKey(clienteID, tenantID)
	carrinho, err := database.GetOrSet(s.cache, ctx, key, 2*time.Minute, func() (*dto.Carrinho, error) {
		return &dto.Carrinho{
			ClienteID: fmt.Sprint(clienteID),
			TenantID:  fmt.Sprint(tenantID),
			Itens:     []dto.ItemCarrinho{},
			Estado:    dto.EstadoAberto,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar carrinho: %w", err)
	}
	// migra carrinhos antigos sem estado
	if carrinho.Estado == "" {
		carrinho.Estado = dto.EstadoAberto
	}
	return carrinho, nil
}

func (s *CarrinhoService) saveCarrinho(ctx context.Context, carrinho *dto.Carrinho) error {
	carrinho.UpdatedAt = time.Now()
	key := s.getKey(parseUint(carrinho.ClienteID), parseUint(carrinho.TenantID))
	if err := s.cache.SetJSONWithContext(ctx, key, carrinho, TTLCarrinho*time.Second); err != nil {
		return fmt.Errorf("erro ao salvar carrinho: %w", err)
	}
	return nil
}

// ProcessarMensagem - DONO DO WORKFLOW com máquina de estados de endereço
func (s *CarrinhoService) ProcessarMensagem(ctx context.Context, clienteID, tenantID uint, input dto.MessageInput) (string, error) {
	logger.Info(
		ctx, "[CARRINHO] Processando mensagem",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("tenant_id", tenantID),
		zap.String("source", string(input.Source)),
	)

	carrinhoAtual, _ := s.GetCarrinho(ctx, clienteID, tenantID)

	// ============================================
	// MÁQUINA DE ESTADOS DE ENDEREÇO - prioridade máxima
	// ============================================
	if carrinhoAtual != nil && carrinhoAtual.Estado != "" && carrinhoAtual.Estado != dto.EstadoAberto {
		switch carrinhoAtual.Estado {
		case dto.EstadoAguardandoEnderecoLista:
			return s.handleSelecaoEndereco(ctx, clienteID, tenantID, carrinhoAtual, input.Text)
		case dto.EstadoAguardandoEnderecoNovo:
			return s.handleNovoEndereco(ctx, clienteID, tenantID, carrinhoAtual, input.Text)
		}
	}

	cardapio, err := s.cardapioService.GetCardapio(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, err.Error())
		return "", fmt.Errorf("erro cardápio: %w", err)
	}

	contextoCarrinho := "carrinho vazio"
	if carrinhoAtual != nil && len(carrinhoAtual.Itens) > 0 {
		nomes := make([]string, 0, len(carrinhoAtual.Itens))
		for _, it := range carrinhoAtual.Itens {
			nomes = append(nomes, it.ProdutoItem.Nome)
		}
		contextoCarrinho = fmt.Sprintf("%d itens: %s", len(carrinhoAtual.Itens), strings.Join(nomes, ", "))
	}

	var intencao *dto.IntencaoCliente

	if len(cardapio) <= limiteCardapioPequeno {
		intencao, err = llm.RetryWithBackoff(ctx, llm.DefaultRetryConfig(), func() (*dto.IntencaoCliente, error) {
			return s.llmService.ResolveItemsByMenu(ctx, tenantID, input, cardapio)
		})
		if err != nil {
			logger.Error(ctx, "Erro ResolveItemsByMenu", zap.Error(err))
			return "⚠ Tive um problema técnico, tenta de novo por favor.", nil
		}
	} else {
		textoBase, err := s.llmService.ObterTextoBase(ctx, tenantID, input)
		if err != nil {
			logger.Error(ctx, "Erro ObterTextoBase", zap.Error(err))
			return "⚠ Não consegui entender seu áudio/imagem, pode digitar?", nil
		}
		if strings.TrimSpace(textoBase) == "" {
			return "Não entendi, pode repetir?", nil
		}

		result, err := s.llmService.ClassificarEExtrairKeywords(ctx, textoBase, contextoCarrinho)
		if err != nil {
			logger.Error(ctx, "Erro ClassificarEExtrairKeywords", zap.Error(err))
			return "⚠ Tive um problema técnico, tenta de novo por favor.", nil
		}

		switch result.Acao {
		case "conversa":
			if result.Resposta != "" {
				return result.Resposta, nil
			}
			return "Olá! 😊 Como posso ajudar?", nil
		case "listar_categorias":
			return s.cardapioService.ListarCategoriasHumanizado(cardapio), nil
		case "listar_produtos":
			return s.cardapioService.ListarProdutosHumanizado(cardapio, result.Filtro), nil
		case "ver_carrinho":
			return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)
		case "limpar_carrinho":
			_ = s.LimparCarrinho(ctx, clienteID, tenantID)
			return "🗑️ Carrinho limpo!", nil
		case "remover":
			if len(result.Keywords) == 0 {
				return "O que você quer remover?", nil
			}
			cardapioReduzido, _ := s.cardapioService.ReduzirPorKeywords(ctx, tenantID, result.Keywords)
			itensRemover, _ := s.llmService.ResolverItensByKeyWords(ctx, tenantID, result.Keywords, cardapioReduzido)
			for _, it := range itensRemover {
				_ = s.RemoverItem(ctx, clienteID, tenantID, it, it.Quantidade)
			}
			return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)
		case "finalizar", "confirmar", "fechar", "checkout":
			// NOVO FLUXO - vai para máquina de estados de endereço
			return s.iniciarFluxoFinalizacao(ctx, clienteID, tenantID)
		default:
			cardapioReduzido, _ := s.cardapioService.ReduzirPorKeywords(ctx, tenantID, result.Keywords)
			intencaoItens, _ := s.llmService.ResolverItensByKeyWords(ctx, tenantID, result.Keywords, cardapioReduzido)
			intencao = &dto.IntencaoCliente{
				Acao:  result.Acao,
				Itens: intencaoItens,
			}
		}
	}

	// fluxo pequeno e fallback do grande
	if intencao == nil {
		return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)
	}

	switch intencao.Acao {
	case "ver_carrinho":
		return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)
	case "limpar_carrinho":
		_ = s.LimparCarrinho(ctx, clienteID, tenantID)
		return "🗑️ Carrinho limpo!", nil
	case "remover":
		if len(intencao.Itens) == 0 {
			return "O que você quer remover?", nil
		}
		for _, it := range intencao.Itens {
			_ = s.RemoverItem(ctx, clienteID, tenantID, it, it.Quantidade)
		}
		return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)
	case "finalizar", "confirmar", "fechar", "checkout":
		return s.iniciarFluxoFinalizacao(ctx, clienteID, tenantID)
	default:
		if len(intencao.Itens) == 0 {
			return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)
		}
		carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
		if err != nil {
			return "", err
		}
		for _, item := range intencao.Itens {
			carrinho = s.mergeItem(carrinho, item)
		}
		// garante que estado volta para aberto ao adicionar item novo
		carrinho.Estado = dto.EstadoAberto
		carrinho.EnderecoID = nil
		if err := s.saveCarrinho(ctx, carrinho); err != nil {
			return "", err
		}
		return s.FormatResumoCarrinho(ctx, carrinho)
	}
}

// ============================================
// FLUXO DE ENDEREÇO - NOVO
// ============================================

func (s *CarrinhoService) iniciarFluxoFinalizacao(ctx context.Context, clienteID, tenantID uint) (string, error) {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return "", err
	}
	if len(carrinho.Itens) == 0 {
		return "🛒 Seu carrinho está vazio. Me diga o que quer adicionar!", nil
	}

	enderecos, err := s.clienteService.ListarEnderecos(ctx, clienteID)
	if err != nil {
		logger.Error(ctx, "erro ao listar endereços", zap.Error(err))
		// falha no banco não bloqueia - pede endereço novo
		enderecos = []dto.EnderecoDTO{}
	}

	if len(enderecos) == 0 {
		// SEM ENDEREÇO - solicita novo
		carrinho.Estado = dto.EstadoAguardandoEnderecoNovo
		carrinho.TentativasEndereco = 0
		_ = s.saveCarrinho(ctx, carrinho)
		return helpers.FormatSolicitarNovoEndereco(false), nil
	}

	// TEM ENDEREÇOS - lista para escolha
	carrinho.Estado = dto.EstadoAguardandoEnderecoLista
	carrinho.TentativasEndereco = 0
	_ = s.saveCarrinho(ctx, carrinho)
	return helpers.FormatListaEnderecos(enderecos), nil
}

func (s *CarrinhoService) handleSelecaoEndereco(ctx context.Context, clienteID, tenantID uint, carrinho *dto.Carrinho, texto string) (string, error) {
	textoLimpo := strings.TrimSpace(strings.ToLower(texto))

	// comando "novo" -> vai para cadastro
	if textoLimpo == "novo" || textoLimpo == "novo endereço" || textoLimpo == "novo endereco" || textoLimpo == "outro" || textoLimpo == "cadastrar" {
		carrinho.Estado = dto.EstadoAguardandoEnderecoNovo
		_ = s.saveCarrinho(ctx, carrinho)
		return helpers.FormatSolicitarNovoEndereco(true), nil
	}

	// tenta parsear número 1,2,3
	if idx, ok := parseIndiceEndereco(textoLimpo); ok {
		enderecos, err := s.clienteService.ListarEnderecos(ctx, clienteID)
		if err != nil || idx < 1 || idx > len(enderecos) {
			carrinho.TentativasEndereco++
			_ = s.saveCarrinho(ctx, carrinho)
			if carrinho.TentativasEndereco >= 2 {
				// após 2 erros, re-lista
				return helpers.FormatListaEnderecos(enderecos), nil
			}
			return fmt.Sprintf("⚠️ Opção inválida. Digite um número de 1 a %d ou *novo*", len(enderecos)), nil
		}
		escolhido := enderecos[idx-1]
		return s.finalizarComEndereco(ctx, clienteID, tenantID, carrinho, escolhido.ID)
	}

	// se parece endereço (tem número e texto longo), trata como novo endereço direto
	if pareceEndereco(texto) {
		return s.handleNovoEndereco(ctx, clienteID, tenantID, carrinho, texto)
	}

	// não entendeu - re-lista
	enderecos, _ := s.clienteService.ListarEnderecos(ctx, clienteID)
	return helpers.FormatListaEnderecos(enderecos), nil
}

func (s *CarrinhoService) handleNovoEndereco(ctx context.Context, clienteID, tenantID uint, carrinho *dto.Carrinho, texto string) (string, error) {
	texto = strings.TrimSpace(texto)
	if len(texto) < 10 {
		return helpers.FormatErroEndereco("Endereço muito curto. Preciso de rua e número."), nil
	}

	// parse simples do endereço - não inventa, só quebra por vírgula
	req := parseEnderecoTexto(texto)
	if req.Logradouro == "" || req.Numero == "" {
		return helpers.FormatErroEndereco("Não consegui identificar rua e número. Envie no formato: *Rua, Número, Bairro*"), nil
	}

	// CRIA - regra de negócio: nunca edita/deleta, só adiciona
	novoEndereco, err := s.clienteService.AdicionarEndereco(ctx, clienteID, req)
	if err != nil {
		logger.Error(ctx, "erro ao criar endereço", zap.Error(err))
		return "❌ Erro ao salvar endereço. Tente novamente no formato: *Rua das Flores, 123, Centro*", nil
	}

	// confirma e já finaliza pedido com ele
	msgConfirm := helpers.FormatEnderecoCadastrado(novoEndereco) + "\n\n"
	pedidoMsg, err := s.finalizarComEndereco(ctx, clienteID, tenantID, carrinho, novoEndereco.ID)
	if err != nil {
		return "", err
	}
	return msgConfirm + pedidoMsg, nil
}

func (s *CarrinhoService) finalizarComEndereco(ctx context.Context, clienteID, tenantID uint, carrinho *dto.Carrinho, enderecoID uint) (string, error) {
	// valida se endereço pertence ao cliente e está ativo
	enderecos, err := s.clienteService.ListarEnderecos(ctx, clienteID)
	if err != nil {
		return "", err
	}
	var encontrado *dto.EnderecoDTO
	for _, e := range enderecos {
		if e.ID == enderecoID {
			encontrado = &e
			break
		}
	}
	if encontrado == nil {
		// pode ser o recém criado que ainda não está na lista cacheada - busca direto
		// tenta finalizar mesmo assim, o repo vai validar FK
		logger.Warn(ctx, "endereço não encontrado na listagem, tentando finalizar mesmo assim", zap.Uint("endereco_id", enderecoID))
	}

	// pega nome do cliente para pedido
	clienteDTO, _ := s.clienteService.FindByID(ctx, clienteID)
	nomeCliente := ""
	if clienteDTO != nil {
		nomeCliente = clienteDTO.Nome
		if nomeCliente == "" {
			nomeCliente = clienteDTO.NomePerfil
		}
	}

	// limpa estado de endereço antes de finalizar para não ficar em loop se falhar
	carrinho.Estado = dto.EstadoAberto
	carrinho.EnderecoID = &enderecoID
	_ = s.saveCarrinho(ctx, carrinho)

	pedidoConfirmado, err := s.FinalizarCarrinhoComEndereco(ctx, clienteID, tenantID, nomeCliente, enderecoID)
	if err != nil {
		if err.Error() == "carrinho vazio" {
			return "🛒 Seu carrinho está vazio. Me diga o que quer adicionar!", nil
		}
		return "❌ Erro ao finalizar pedido.", nil
	}

	// limpa carrinho já é feito dentro de FinalizarCarrinhoComEndereco
	return s.FormatarPedidoConfirmado(pedidoConfirmado), nil
}

// ============================================
// MÉTODOS EXISTENTES - mantidos + novo FinalizarComEndereco
// ============================================

func (s *CarrinhoService) mergeItem(carrinho *dto.Carrinho, item dto.ItemCarrinho) *dto.Carrinho {
	for i, existing := range carrinho.Itens {
		if existing.ProdutoItem.ID == item.ProdutoItem.ID {
			carrinho.Itens[i].Quantidade += item.Quantidade
			if item.Observacao != "" {
				carrinho.Itens[i].Observacao = item.Observacao
			}
			return carrinho
		}
	}
	carrinho.Itens = append(carrinho.Itens, item)
	return carrinho
}

func (s *CarrinhoService) AdicionarItem(ctx context.Context, clienteID, tenantID uint, item dto.ItemCarrinho) error {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return err
	}
	carrinho = s.mergeItem(carrinho, item)
	return s.saveCarrinho(ctx, carrinho)
}

func (s *CarrinhoService) RemoverItem(ctx context.Context, clienteID, tenantID uint, itemCarrinho dto.ItemCarrinho, quantidade int) error {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return err
	}
	for i, item := range carrinho.Itens {
		if item.ProdutoItem.ID == itemCarrinho.ProdutoItem.ID {
			if quantidade == 0 || quantidade >= item.Quantidade {
				carrinho.Itens = append(carrinho.Itens[:i], carrinho.Itens[i+1:]...)
			} else {
				carrinho.Itens[i].Quantidade -= quantidade
			}
			return s.saveCarrinho(ctx, carrinho)
		}
	}
	return fmt.Errorf("item '%s' não encontrado", itemCarrinho.ProdutoItem.Nome)
}

func (s *CarrinhoService) LimparCarrinho(ctx context.Context, clienteID, tenantID uint) error {
	key := s.getKey(clienteID, tenantID)
	return s.cache.DeleteWithContext(ctx, key)
}

func (s *CarrinhoService) CalcularTotal(carrinho *dto.Carrinho) float64 {
	total := 0.0
	for _, item := range carrinho.Itens {
		total += item.Preco * float64(item.Quantidade)
	}
	return total
}

func (s *CarrinhoService) CalcularTempoEstimado(carrinho *dto.Carrinho) int {
	if len(carrinho.Itens) == 0 {
		return 0
	}
	total := 0
	for _, item := range carrinho.Itens {
		total += item.Quantidade
	}
	return 15 + (total * 5)
}

// FinalizarCarrinho - mantido para compatibilidade, finaliza SEM endereço (retirada)
func (s *CarrinhoService) FinalizarCarrinho(ctx context.Context, clienteID, tenantID uint, clienteNome string) (*dto.PedidoConfirmado, error) {
	return s.FinalizarCarrinhoComEndereco(ctx, clienteID, tenantID, clienteNome, 0)
}

// FinalizarCarrinhoComEndereco - NOVO - com endereço de entrega
func (s *CarrinhoService) FinalizarCarrinhoComEndereco(ctx context.Context, clienteID, tenantID uint, clienteNome string, enderecoID uint) (*dto.PedidoConfirmado, error) {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return nil, err
	}
	if len(carrinho.Itens) == 0 {
		return nil, fmt.Errorf("carrinho vazio")
	}
	pedidoExtraido := &dto.PedidoExtraido{}
	for _, item := range carrinho.Itens {
		pedidoExtraido.Itens = append(pedidoExtraido.Itens, dto.ItemPedidoInput{
			ProdutoItem:   item.ProdutoItem,
			Quantidade:    item.Quantidade,
			Observacao:    item.Observacao,
			PrecoUnitario: item.Preco,
		})
	}
	var enderecoPtr *uint
	if enderecoID != 0 {
		enderecoPtr = &enderecoID
	}
	confirmado, err := s.pedidoService.ProcessarPedidoComEndereco(ctx, tenantID, clienteID, clienteNome, pedidoExtraido, enderecoPtr)
	if err != nil {
		return nil, err
	}
	_ = s.LimparCarrinho(ctx, clienteID, tenantID)
	return confirmado, nil
}

func (s *CarrinhoService) BuscarProdutos(ctx context.Context, tenantID, termo string, limit int) ([]dto.ProdutoItem, error) {
	return s.produtoRepo.BuscarProdutosPorNome(ctx, tenantID, termo, limit)
}

func (s *CarrinhoService) BuscarProdutosLote(ctx context.Context, tenantID string, nomes []string) (map[string]dto.ProdutoItem, error) {
	return s.produtoRepo.BuscarProdutosLote(ctx, tenantID, nomes)
}

func (s *CarrinhoService) FormatResumoCarrinhoByCliente(ctx context.Context, clienteID, tenantID uint) (string, error) {
	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return "", err
	}
	return s.FormatResumoCarrinho(ctx, carrinho)
}

func (s *CarrinhoService) FormatResumoCarrinho(ctx context.Context, carrinho *dto.Carrinho) (string, error) {
	total := s.CalcularTotal(carrinho)
	tempo := s.CalcularTempoEstimado(carrinho)
	return helpers.FormatResumoCarrinho(carrinho.Itens, total, tempo), nil
}

func (s *CarrinhoService) FormatarPedidoConfirmado(pedido *dto.PedidoConfirmado) string {
	return helpers.FormatRespostaPedido(pedido)
}

// ============================================
// HELPERS PRIVADOS - parsing de endereço
// ============================================

func parseUint(s string) uint {
	var v uint
	fmt.Sscan(s, &v)
	return v
}

func parseIndiceEndereco(texto string) (int, bool) {
	texto = strings.TrimSpace(texto)
	// aceita 1, 1., 1) , opção 1
	re := regexp.MustCompile(`^(\d+)[\.\)]?$`)
	matches := re.FindStringSubmatch(texto)
	if len(matches) < 2 {
		// tenta extrair primeiro número
		re2 := regexp.MustCompile(`\d+`)
		numStr := re2.FindString(texto)
		if numStr == "" {
			return 0, false
		}
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	n, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

func pareceEndereco(texto string) bool {
	textoLower := strings.ToLower(texto)
	if len(texto) < 12 {
		return false
	}
	// tem número e tem indicador de rua/av/bairro
	temNumero := regexp.MustCompile(`\d+`).MatchString(texto)
	temIndicador := strings.Contains(textoLower, "rua") || strings.Contains(textoLower, "av") || strings.Contains(textoLower, "bairro") || strings.Contains(textoLower, ",") || strings.Contains(textoLower, "nº") || strings.Contains(textoLower, "numero")
	return temNumero && temIndicador
}

func parseEnderecoTexto(texto string) *dto.CriarEnderecoRequest {
	// Formato esperado: Rua das Flores, 123, Centro, Pinhalzinho - SC, 89870-000
	// Quebra por vírgula - simples e robusto
	partes := strings.Split(texto, ",")
	for i := range partes {
		partes[i] = strings.TrimSpace(partes[i])
	}

	req := &dto.CriarEnderecoRequest{
		Pais:      "Brasil",
		Tipo:      "entrega",
		Principal: false,
	}

	if len(partes) >= 1 {
		req.Logradouro = partes[0]
	}
	if len(partes) >= 2 {
		// segunda parte geralmente é número + bairro ou só número
		// tenta extrair número
		reNum := regexp.MustCompile(`^(\d+\w*)\s*(.*)`)
		m := reNum.FindStringSubmatch(partes[1])
		if len(m) >= 2 {
			req.Numero = m[1]
			if len(m) >= 3 && m[2] != "" {
				req.Bairro = m[2]
			}
		} else {
			req.Numero = partes[1]
		}
	}
	if len(partes) >= 3 {
		// se bairro ainda vazio, terceira parte é bairro
		if req.Bairro == "" {
			req.Bairro = partes[2]
		} else {
			// terceira parte pode ser cidade ou complemento
			req.Cidade = partes[2]
		}
	}
	if len(partes) >= 4 {
		// tenta extrair cidade e estado: Pinhalzinho - SC
		cidadeEstado := partes[3]
		if strings.Contains(cidadeEstado, "-") {
			ce := strings.Split(cidadeEstado, "-")
			req.Cidade = strings.TrimSpace(ce[0])
			if len(ce) > 1 {
				req.Estado = strings.TrimSpace(ce[1])
			}
		} else {
			req.Cidade = cidadeEstado
		}
	}
	if len(partes) >= 5 {
		// CEP
		cepRe := regexp.MustCompile(`\d{5}-?\d{3}`)
		cep := cepRe.FindString(partes[4])
		if cep != "" {
			req.CEP = cep
		}
	}

	// fallback: extrai CEP de qualquer lugar do texto
	if req.CEP == "" {
		cepRe := regexp.MustCompile(`\d{5}-?\d{3}`)
		req.CEP = cepRe.FindString(texto)
	}

	// fallback: se não conseguiu separar número, tenta regex geral
	if req.Numero == "" {
		re := regexp.MustCompile(`(?i)(?:n[º°]?|numero)\s*(\d+)`)
		if m := re.FindStringSubmatch(texto); len(m) > 1 {
			req.Numero = m[1]
		} else {
			// pega primeiro número isolado após logradouro
			re2 := regexp.MustCompile(`\b(\d{1,5})\b`)
			nums := re2.FindAllString(texto, -1)
			if len(nums) > 0 {
				// ignora CEP
				for _, n := range nums {
					if len(n) <= 5 && n != strings.ReplaceAll(req.CEP, "-", "")[:5] {
						req.Numero = n
						break
					}
				}
			}
		}
	}

	if req.Numero == "" {
		req.Numero = "S/N"
	}

	// se logradouro ainda contém número, limpa
	req.Logradouro = strings.TrimSpace(req.Logradouro)

	return req
}
