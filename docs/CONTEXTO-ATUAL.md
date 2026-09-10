
# CONTEXTO - mcp-server-openerp - Workflow Novo

## Objetivo
Resolver bug WhatsApp reenviando mensagens + criar workflow limpo carrinho/pedido com validação por ID MySQL, sem quebrar front.

## Arquitetura Nova

### Webhook -> handler
- webhook recebe WhatsApp (dedup 5min por messageID via cache)
- responde 200 rápido pra parar reenvio
- cria dto.MessageInput { Source, Text, Audio []byte, Image []byte } - só bytes, sem transcrição
- chama carrinhoService.ProcessarMensagem

### llm/provider.go (DONO DO FLUXO)
Arquivo final em /mnt/data/provider.go - INTACTO

type UnifiedLLM struct {
  pool map[string]LLMClient
  textClient TextLLM
  audioClient AudioLLM
  visionClient VisionLLM
  preprocessor *Preprocessor
}

Fluxo ExtractIntent:
1. switch input.Source:
   - audio -> TranscribeAudio via audioClient
   - image -> DescribeImage via visionClient
   - text -> texto direto
2. preprocessor.Process(textoBase) -> textoLimpo
3. keywords via provider da origem (audio->audioClient, image->visionClient, text->textClient)
   prompt: PromptExtractKeywords
   retorna: { keywords: []string }
4. Se keywords vazio -> Acao=visualizar
5. Resolve IDs SEMPRE via textClient:
   lista = formatCardapioForPrompt (formato "ID - Nome")
   prompt: PromptResolverIDs (lista, keywords, lista)
   retorna: { itens: [{id, qtd, obs}] }
6. mapResolverToCarrinho: valida ID no mapCardapio MySQL, qtd<=0 =>1, monta []ItemCarrinho{ProdutoItem, Quantidade, Observacao}
7. Retorna IntencaoCliente{Acao: adicionar, Itens, Mensagem: textoBase original}

### carrinho_service.go
ProcessarMensagem é único entry point
- GetCardapio 1h cache
- ExtractIntent com RetryWithBackoff (429)
- Otimizado: 1 GetCarrinho + merge memória + 1 SaveCarrinho (antes era N+1)
- RemoverItem bug ID==ID corrigido
- saveCarrinho(ctx) não Background
- FinalizarCarrinho: ItemCarrinho -> ItemPedidoInput com preço MySQL

### cardapio_service.go
- Usa cache.Cache (não redis.Client cru)
- Métodos públicos MANTIDOS pro front:
  - GetCardapio(ctx, tenantID uint)
  - BuscarProdutoPorNome(ctx, tenantID string, nome string) - assinatura string pra não quebrar front
  - ItemExisteNoCardapio(cardapio, nome) (*ProdutoItem, error)
  - EncontrarItemSimilar(cardapio, nome) string
  - FormatarCardapio(cardapio) string
  - FindByID, ListWithFilters

### pedido_service.go - INTERFACE RESPEITADA
type PedidoServiceInterface interface {
  ProcessarPedido(ctx, tenantID, clienteID uint, clienteNome string, pedidoExtraido *dto.PedidoExtraido) (*dto.PedidoConfirmado, error)
  FindByTenant(ctx, tenantID uint) ([]dto.PedidoDTO, error)
  CountPedidosHoje(ctx, tenantID uint) (int64, error)
  CountPedidosSemana(ctx, tenantID uint) (int64, error)
  CountPorStatus(ctx, tenantID uint) (map[string]int64, error)
  CountPendentes(ctx, tenantID uint) (int64, error)
  FaturamentoHoje(ctx, tenantID uint) (float64, error)
  FaturamentoMes(ctx, tenantID uint) (float64, error)
  ListWithFilters(ctx, tenantID, clienteID uint, status string, dataInicio, dataFim time.Time, page, limit int) ([]dto.PedidoDTO, int64, error)
  FindByID(ctx, id uint) (*dto.PedidoDTO, error)
  ListByCliente(ctx, clienteID uint, page, limit int) ([]dto.PedidoDTO, int64, error)
  AtualizarStatusPedido(ctx, id uint, status string) (*dto.PedidoDTO, error)
  Create(ctx, req *dto.CriarPedidoRequest) (*dto.PedidoDTO, error)
}

Implementação:
- ProcessarPedido: corrigido bug ItemExisteNoCardapio retornava (*ProdutoItem, error) mas código tratava como (bool, float) e variável preco indefinida
- Usa cardapio validado, fallback similar
- Dashboard delega pra repo: CountByPeriodo, CountGroupByStatus, CountByStatus, SumTotalByPeriodo, FindByTenantPeriodo
- Create implementado

### dto
type ItemPedidoDTO { Nome, Quantidade, Preco, Observacao }
type CriarPedidoRequest { TenantID, ClienteID, EnderecoEntregaID, Itens []ItemPedidoDTO, Observacoes, Origem }
+ método Total() float32 e TotalFloat64() float64

## Arquivos Finais Corrigidos (texto puro enviado, não artefato)
- provider.go intacto final está em /mnt/data/provider.go
- pedido_service.go corrigido enviado em texto na conversa (respeitando interface)
- cardapio_service.go corrigido enviado em texto (mantendo métodos front)
- carrinho_service.go otimizado (versão clean)

## Regra de Ouro Acordada
- NUNCA remover método público sem avaliar front - interface é contrato
- Sempre respeitar assinaturas da interface

## Próximos Passos
- Commitar tudo com mensagem gerada
- Iniciar nova conversa com este contexto

## Commit Message Gerada
feat: novo workflow carrinho/pedido com validação por ID MySQL e dedup WhatsApp
... (mensagem completa já gerada na conversa anterior)

## Data
2026-05-13
