// internal/service/carrinho_service.go - FIX FINAL compatível com classifier.go V2 + endereço só no finalizar
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
	"github.com/rafapasa/mcp-server-openerp/internal/intent"
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

func (s *CarrinhoService) ProcessarMensagem(ctx context.Context, clienteID, tenantID uint, input dto.MessageInput) (string, error) {
	logger.Info(
		ctx, "[CARRINHO] Processando mensagem",
		zap.Uint("cliente_id", clienteID),
		zap.Uint("tenant_id", tenantID),
		zap.String("source", string(input.Source)),
	)

	carrinhoAtual, _ := s.GetCarrinho(ctx, clienteID, tenantID)

	// 1. Texto base (audio/imagem -> texto)
	textoBase, err := s.llmService.ObterTextoBase(ctx, tenantID, input)
	if err != nil {
		logger.Error(ctx, "Erro ObterTextoBase", zap.Error(err))
		return "⚠ Não consegui entender seu áudio/imagem, pode digitar?", nil
	}
	textoBase = strings.TrimSpace(textoBase)
	if textoBase == "" {
		return "Não entendi, pode repetir?", nil
	}

	// 2. Se está em fluxo de endereço (só entra aqui DEPOIS de iniciar finalizar)
	if carrinhoAtual != nil && carrinhoAtual.Estado != "" && carrinhoAtual.Estado != dto.EstadoAberto {
		textoLower := strings.ToLower(strings.TrimSpace(textoBase))

		// comandos que cancelam endereço e voltam ao normal
		if textoLower == "limpar carrinho" || textoLower == "limpar" || strings.Contains(textoLower, "cancelar") || textoLower == "cancelar pedido" {
			carrinhoAtual.Estado = dto.EstadoAberto
			carrinhoAtual.EnderecoID = nil
			_ = s.saveCarrinho(ctx, carrinhoAtual)
			if strings.Contains(textoLower, "limpar") {
				_ = s.LimparCarrinho(ctx, clienteID, tenantID)
				return "🗑️ Carrinho limpo!", nil
			}
			return "Checkout cancelado. Seu carrinho continua aqui 👇\n" + mustFormat(s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)), nil
		}

		switch carrinhoAtual.Estado {
		case dto.EstadoAguardandoEnderecoLista:
			if _, ok := parseIndiceEndereco(textoLower); ok || pareceEndereco(textoBase) || textoLower == "novo" || strings.Contains(textoLower, "novo endereço") || strings.Contains(textoLower, "outro") {
				return s.handleSelecaoEndereco(ctx, clienteID, tenantID, carrinhoAtual, textoBase)
			}
			// se não parece endereço, reseta e continua fluxo normal (ex: "quero mais um x")
			carrinhoAtual.Estado = dto.EstadoAberto
			_ = s.saveCarrinho(ctx, carrinhoAtual)
		case dto.EstadoAguardandoEnderecoNovo:
			if len(textoBase) >= 10 {
				return s.handleNovoEndereco(ctx, clienteID, tenantID, carrinhoAtual, textoBase)
			}
		}
	}

	// 3. Fast-path 0 token usando seu classifier V2
	intentRes := intent.ClassifyV2(textoBase, time.Time{})
	switch intentRes.Type {
	case intent.IntentGreeting:
		return intent.GreetingResponse("", time.Now().Hour()), nil
	case intent.IntentGreetingWithAdd:
		// "bom dia quero um x-bacon" -> usa CleanRest como texto pra adicionar
		textoBase = intentRes.CleanRest
		// não retorna, continua pro LLM resolver o resto
	case intent.IntentSmallTalk:
		if resp := intent.SmallTalkResponse(textoBase); resp != "" {
			return resp, nil
		}
	case intent.IntentThanks:
		return intent.ThanksResponse(), nil
	case intent.IntentViewCart:
		return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)
	case intent.IntentClearCart:
		_ = s.LimparCarrinho(ctx, clienteID, tenantID)
		return "🗑️ Carrinho limpo!", nil
	case intent.IntentNone:
		// anti-spam saudação 3min -> ignora
		return "", nil
	}

	// 4. Contexto carrinho para LLM
	contextoCarrinho := "carrinho vazio"
	if carrinhoAtual != nil && len(carrinhoAtual.Itens) > 0 {
		nomes := make([]string, 0, len(carrinhoAtual.Itens))
		for _, it := range carrinhoAtual.Itens {
			nomes = append(nomes, it.ProdutoItem.Nome)
		}
		contextoCarrinho = fmt.Sprintf("%d itens: %s", len(carrinhoAtual.Itens), strings.Join(nomes, ", "))
	}

	// 5. Classifica via LLM ANTES do branch tamanho cardápio - FIX BUG conversa
	result, err := s.llmService.ClassificarEExtrairKeywords(ctx, tenantID, textoBase, contextoCarrinho)
	if err != nil {
		logger.Error(ctx, "Erro ClassificarEExtrairKeywords", zap.Error(err))
		return "⚠ Tive um problema técnico, tenta de novo por favor.", nil
	}

	switch result.Acao {
	case "conversa":
		if result.Resposta != "" {
			return result.Resposta, nil
		}
		return "Olá! 😊 Como posso ajudar? Me diga o que você gostaria de pedir.", nil
	case "listar_categorias", "ver_cardapio", "cardapio":
		cardapio, _ := s.cardapioService.GetCardapio(ctx, tenantID)
		return s.cardapioService.ListarCategoriasHumanizado(cardapio), nil
	case "listar_produtos":
		cardapio, _ := s.cardapioService.GetCardapio(ctx, tenantID)
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
		return s.iniciarFluxoFinalizacao(ctx, clienteID, tenantID)
	}

	// 6. Adicionar item - branch por tamanho
	cardapio, err := s.cardapioService.GetCardapio(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, err.Error())
		return "", fmt.Errorf("erro cardápio: %w", err)
	}

	var intencao *dto.IntencaoCliente

	if len(cardapio) <= limiteCardapioPequeno {
		intencao, err = llm.RetryWithBackoff(ctx, llm.DefaultRetryConfig(), func() (*dto.IntencaoCliente, error) {
			inputTexto := dto.MessageInput{Text: textoBase, Source: input.Source}
			return s.llmService.ResolveItemsByMenu(ctx, tenantID, inputTexto, cardapio)
		})
		if err != nil {
			logger.Error(ctx, "Erro ResolveItemsByMenu", zap.Error(err))
			return "⚠ Tive um problema técnico, tenta de novo por favor.", nil
		}
		if intencao.Acao == "conversa" {
			if intencao.Mensagem != "" {
				return intencao.Mensagem, nil
			}
			return "Olá! 😊 Como posso ajudar?", nil
		}
	} else {
		cardapioReduzido, _ := s.cardapioService.ReduzirPorKeywords(ctx, tenantID, result.Keywords)
		itens, _ := s.llmService.ResolverItensByKeyWords(ctx, tenantID, result.Keywords, cardapioReduzido)
		intencao = &dto.IntencaoCliente{
			Acao:  result.Acao,
			Itens: itens,
		}
	}

	if intencao == nil || len(intencao.Itens) == 0 {
		if carrinhoAtual != nil && len(carrinhoAtual.Itens) > 0 {
			return s.FormatResumoCarrinhoByCliente(ctx, clienteID, tenantID)
		}
		return "Não entendi qual item você quer. Pode me dizer como no cardápio? Ex: *quero 1 X-Bacon*", nil
	}

	carrinho, err := s.GetCarrinho(ctx, clienteID, tenantID)
	if err != nil {
		return "", err
	}
	for _, item := range intencao.Itens {
		carrinho = s.mergeItem(carrinho, item)
	}
	carrinho.Estado = dto.EstadoAberto
	carrinho.EnderecoID = nil
	if err := s.saveCarrinho(ctx, carrinho); err != nil {
		return "", err
	}
	return s.FormatResumoCarrinho(ctx, carrinho)
}

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
		enderecos = []dto.EnderecoDTO{}
	}

	if len(enderecos) == 0 {
		carrinho.Estado = dto.EstadoAguardandoEnderecoNovo
		carrinho.TentativasEndereco = 0
		_ = s.saveCarrinho(ctx, carrinho)
		return helpers.FormatSolicitarNovoEndereco(false), nil
	}

	carrinho.Estado = dto.EstadoAguardandoEnderecoLista
	carrinho.TentativasEndereco = 0
	_ = s.saveCarrinho(ctx, carrinho)
	return helpers.FormatListaEnderecos(enderecos), nil
}

func (s *CarrinhoService) handleSelecaoEndereco(ctx context.Context, clienteID, tenantID uint, carrinho *dto.Carrinho, texto string) (string, error) {
	textoLimpo := strings.TrimSpace(strings.ToLower(texto))

	if textoLimpo == "novo" || strings.Contains(textoLimpo, "novo endereço") || textoLimpo == "outro" || textoLimpo == "cadastrar" {
		carrinho.Estado = dto.EstadoAguardandoEnderecoNovo
		_ = s.saveCarrinho(ctx, carrinho)
		return helpers.FormatSolicitarNovoEndereco(true), nil
	}

	if idx, ok := parseIndiceEndereco(textoLimpo); ok {
		enderecos, err := s.clienteService.ListarEnderecos(ctx, clienteID)
		if err != nil || idx < 1 || idx > len(enderecos) {
			carrinho.TentativasEndereco++
			_ = s.saveCarrinho(ctx, carrinho)
			if carrinho.TentativasEndereco >= 2 {
				return helpers.FormatListaEnderecos(enderecos), nil
			}
			return fmt.Sprintf("⚠️ Opção inválida. Digite um número de 1 a %d ou *novo*", len(enderecos)), nil
		}
		escolhido := enderecos[idx-1]
		return s.finalizarComEndereco(ctx, clienteID, tenantID, carrinho, escolhido.ID)
	}

	if pareceEndereco(texto) {
		return s.handleNovoEndereco(ctx, clienteID, tenantID, carrinho, texto)
	}

	enderecos, _ := s.clienteService.ListarEnderecos(ctx, clienteID)
	return helpers.FormatListaEnderecos(enderecos), nil
}

func (s *CarrinhoService) handleNovoEndereco(ctx context.Context, clienteID, tenantID uint, carrinho *dto.Carrinho, texto string) (string, error) {
	texto = strings.TrimSpace(texto)
	if len(texto) < 10 {
		return helpers.FormatErroEndereco("Endereço muito curto. Preciso de rua e número."), nil
	}

	req := parseEnderecoTexto(texto)
	if req.Logradouro == "" || req.Numero == "" {
		return helpers.FormatErroEndereco("Não consegui identificar rua e número. Envie no formato: *Rua, Número, Bairro*"), nil
	}

	novoEndereco, err := s.clienteService.AdicionarEndereco(ctx, clienteID, req)
	if err != nil {
		logger.Error(ctx, "erro ao criar endereço", zap.Error(err))
		return "❌ Erro ao salvar endereço. Tente novamente no formato: *Rua das Flores, 123, Centro*", nil
	}

	msgConfirm := helpers.FormatEnderecoCadastrado(novoEndereco) + "\n\n"
	pedidoMsg, err := s.finalizarComEndereco(ctx, clienteID, tenantID, carrinho, novoEndereco.ID)
	if err != nil {
		return "", err
	}
	return msgConfirm + pedidoMsg, nil
}

func (s *CarrinhoService) finalizarComEndereco(ctx context.Context, clienteID, tenantID uint, carrinho *dto.Carrinho, enderecoID uint) (string, error) {
	clienteDTO, _ := s.clienteService.FindByID(ctx, clienteID)
	nomeCliente := ""
	if clienteDTO != nil {
		nomeCliente = clienteDTO.Nome
		if nomeCliente == "" {
			nomeCliente = clienteDTO.NomePerfil
		}
	}

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

	_ = s.clienteService.AtualizarUltimoPedido(ctx, clienteID)

	return s.FormatarPedidoConfirmado(pedidoConfirmado), nil
}

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

func (s *CarrinhoService) FinalizarCarrinho(ctx context.Context, clienteID, tenantID uint, clienteNome string) (*dto.PedidoConfirmado, error) {
	return s.FinalizarCarrinhoComEndereco(ctx, clienteID, tenantID, clienteNome, 0)
}

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

func parseUint(s string) uint {
	var v uint
	fmt.Sscan(s, &v)
	return v
}

func mustFormat(s string, _ error) string { return s }

func parseIndiceEndereco(texto string) (int, bool) {
	texto = strings.TrimSpace(texto)
	re := regexp.MustCompile(`^(\d+)[\.\)]?$`)
	matches := re.FindStringSubmatch(texto)
	if len(matches) < 2 {
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
	temNumero := regexp.MustCompile(`\d+`).MatchString(texto)
	temIndicador := strings.Contains(textoLower, "rua") || strings.Contains(textoLower, "av") || strings.Contains(textoLower, "bairro") || strings.Contains(textoLower, ",") || strings.Contains(textoLower, "nº") || strings.Contains(textoLower, "numero")
	return temNumero && temIndicador
}

func parseEnderecoTexto(texto string) *dto.CriarEnderecoRequest {
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
		if req.Bairro == "" {
			req.Bairro = partes[2]
		} else {
			req.Cidade = partes[2]
		}
	}
	if len(partes) >= 4 {
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
		cepRe := regexp.MustCompile(`\d{5}-?\d{3}`)
		cep := cepRe.FindString(partes[4])
		if cep != "" {
			req.CEP = cep
		}
	}

	if req.CEP == "" {
		cepRe := regexp.MustCompile(`\d{5}-?\d{3}`)
		req.CEP = cepRe.FindString(texto)
	}

	if req.Numero == "" {
		re := regexp.MustCompile(`(?i)(?:n[º°]?|numero)\s*(\d+)`)
		if m := re.FindStringSubmatch(texto); len(m) > 1 {
			req.Numero = m[1]
		} else {
			re2 := regexp.MustCompile(`\b(\d{1,5})\b`)
			nums := re2.FindAllString(texto, -1)
			if len(nums) > 0 {
				for _, n := range nums {
					if len(n) <= 5 {
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

	req.Logradouro = strings.TrimSpace(req.Logradouro)
	return req
}
