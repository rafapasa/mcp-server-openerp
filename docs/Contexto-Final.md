# 📄 Contexto - mcp-server-openerp

> Documento único de contexto para devs e IAs. Junta README, AGENTS, GIT_WORKFLOW, CHANGELOG e a evolução do fluxo de endereço de 29/08/2026.

**Data:** 29/08/2026  
**Autor:** Rafael Pasa  
**Stack:** Go 1.22+ / Fiber v2 / MySQL + Redis / Wire / WhatsApp Cloud API / Multi-LLM (Groq Whisper, Gemini Vision, DeepSeek)  
**Status:** Produção na OCI - fluxo de endereço implementado

---

## 1. Visão Geral

Servidor de pedidos multi-setor (restaurante, mercado, farmácia) via WhatsApp. O cliente fala em texto, áudio, imagem ou PDF e o sistema interpreta via LLM, gerencia carrinho no Redis e finaliza pedido no MySQL com endereço de entrega.

**Regra de ouro:** Quem chama repositório é só o seu próprio service. `PedidoService` nunca chama `clienteRepo` ou `enderecoRepo`.

---

## 2. Stack

- **Framework:** Fiber v2 (fasthttp) - zero `net/http` em handlers
- **DI:** Google Wire (`make wire` gera `wire_gen.go`)
- **Cache:** Redis com `GetOrSet[T]` genérico - elimina 80% MySQL
- **Banco:** MySQL 8.4 + GORM + Soft Delete em endereços
- **LLMs:**
  - Groq Whisper `whisper-large-v3` - áudio -> texto
  - Gemini `gemini-2.0-flash` - imagem/receita/PDF -> JSON
  - DeepSeek `deepseek-chat` - extração de intenção e itens
- **Obs:** Zap logger, Prometheus metrics, OTEL tracing

---

## 3. Estrutura Atual

```
cmd/
  server/main.go  # Fiber HTTP único (webhook + api dashboard)
  stdio/main.go   # MCP stdio para inspector
  migrate/main.go # GORM migrate

internal/
  di/             # Wire: provider.go, wire.go, wire_gen.go
  config/         # Viper
  database/       # mysql.go, redis.go, interface.go (RedisInterface), migrate.go, seed.go
  dto/
    carrinho.go   # Carrinho com máquina de estados de endereço
    pedido.go     # MERGE de pedido.go + pedido_dto.go - PedidoDTO, ItemPedidoDTO, PedidoConfirmado com EnderecoEntrega
    cliente_dto.go # ClienteDTO, EnderecoDTO, CriarEnderecoRequest
    produto.go
    whatsapp/     # webhook payload
  helpers/
    formatter.go  # FormatResumoCarrinho, FormatRespostaPedido (com endereço), FormatListaEnderecos, FormatSolicitarNovoEndereco
  models/
    cliente.go    # Cliente + Endereco (Principal bool, DeletedAt)
    pedido.go     # Pedido com EnderecoEntregaID *uint + EnderecoEntrega relation
    tenant.go, produto.go, categoria.go, users.go
  repository/
    cliente_repo.go, endereco_repo.go (Create, FindByID, FindByCliente, FindByClienteAtivos - Update/Delete existem mas fluxo novo só usa Create), pedido_repo.go, produto_repo.go, tenant_repo.go
  service/
    interface.go  # TODAS interfaces - CarrinhoServiceInterface agora tem FinalizarCarrinhoComEndereco
    carrinho_service.go # DONO DO WORKFLOW + máquina de estados de endereço
    pedido_service.go   # Só chama pedidoRepo + cardapioService - ProcessarPedidoComEndereco recebe *uint enderecoID
    cliente_service.go  # AdicionarEndereco (só Create, nunca edita/deleta), ListarEnderecos, AtualizarUltimoPedido
    cardapio_service.go
    llm_service.go
  llm/
    provider.go   # UnifiedLLM - DONO DO FLUXO LLM - TranscribeAudio, DescribeImage, ExtractKeywords, ResolveItemsByMenu
    preprocessor.go # Único lugar que sanitiza texto (2 etapas)
    prompts.go, retry.go
  webhook/
    handlers.go   # Cria dto.MessageInput{Source, Text, Audio[], Image[]} e chama carrinhoService.ProcessarMensagem
    whatsapp.go   # WhatsAppClient + dedup 5min por messageID + 200 rápido
    processor.go
    security.go   # Verify HMAC
  server/
    http_server.go  # Sobe Fiber, registra /webhook e /api/v1
    api_handlers.go # Dashboard - só chama services, nunca repo direto
    middleware/auth.go
    icons.go
  observability/
    logger, metrics, middleware (logging, ratelimit, security headers), tracing, health

docs/
  Contexto.md     # este arquivo
  GIT_WORKFLOW.md
  AGENTS.md
  changelog.md
```

---

## 4. Fluxo de Mensagem - Atualizado 29/08/2026 com Endereço

```
WhatsApp POST /webhook
  ↓
whatsapp.go: dedup 5min via Redis (messageID) + responde 200 <100ms
  ↓
handlers.go: monta MessageInput{Source: text|audio|image, Text, Audio []byte, Image []byte}
  ↓
carrinho_service.go ProcessarMensagem (ÚNICO entry point)
  ├─ Se Estado != aberto:
  │   ├─ aguardando_endereco_selecao -> handleSelecaoEndereco (parse "1", "novo", ou texto que parece endereço)
  │   └─ aguardando_endereco_novo   -> handleNovoEndereco (parseEnderecoTexto por vírgula, Create via clienteService)
  │
  ├─ GetCarrinho cache Redis (TTL 3600s)
  ├─ cardapioService.GetCardapio cache 1h
  ├─ intent.Classify (greeting 0 token) -> fast-path
  ├─ llmService.ObterTextoBase (audio->Groq, image->Gemini)
  ├─ llmService.ClassificarEExtrairKeywords + ResolverItensByKeyWords
  ├─ mergeItem + saveCarrinho 1x
  └─ finalizar -> iniciarFluxoFinalizacao:
        ListarEnderecos via clienteService
        0 endereços -> EstadoAguardandoEnderecoNovo + FormatSolicitarNovoEndereco
        N endereços -> EstadoAguardandoEnderecoLista + FormatListaEnderecos
        escolha -> FinalizarCarrinhoComEndereco -> pedidoService.ProcessarPedidoComEndereco(enderecoID) -> LimparCarrinho + AtualizarUltimoPedido via clienteService
  ↓
WhatsApp SendMessage (FormatResumoCarrinho / FormatRespostaPedido com endereço)
```

**Mídias:**
- `audio/voice` -> DownloadMedia -> Groq Whisper -> texto
- `image` -> base64 -> Gemini Vision -> JSON
- `document/pdf` -> Gemini Vision -> texto

---

## 5. Regras de Negócio - Endereço (29/08/2026)

1. **Imutabilidade:** Endereços nunca são editados ou deletados, só adiciona novo. `Update`/`Delete` do repo existem mas não são usados no fluxo WhatsApp.
2. **Soft Delete:** Model `Endereco` tem `DeletedAt gorm.DeletedAt` - `FindByClienteAtivos` filtra `deleted_at IS NULL`.
3. **Fluxo de finalização:**
   - Cliente diz `finalizar` -> verifica endereços
   - Se tem -> lista `*1* - Rua, 123 ⭐ Principal` e pede número ou `novo`
   - Se não tem -> solicita endereço completo `Rua das Flores, 123, Centro, Pinhalzinho - SC, 89870-000`
   - Parse por vírgula: `Logradouro, Numero, Bairro, Cidade - UF, CEP` + regex CEP `\d{5}-?\d{3}` + fallback `S/N`
   - `clienteService.AdicionarEndereco` (Create) -> finaliza pedido com `EnderecoEntregaID`
4. **Pedido:** `models.Pedido` tem `EnderecoEntregaID *uint` nullable (retirada = nil) + relação `EnderecoEntrega`.
5. **DTO Merge:** `internal/dto/pedido.go` é único agora - merge de `pedido.go` + `pedido_dto.go`. Deletar `pedido_dto.go`. Campos duplicados mesclados: `ItemPedidoDTO` tem `ProdutoID *uint` opcional.

---

## 6. Obrigações por Arquivo

| Arquivo | Dono | Nunca pode |
|---------|------|------------|
| `webhook/whatsapp.go` | Receber POST Meta, dedup 5min, 200 rápido, extrair Source | Chamar LLM, MySQL |
| `webhook/handlers.go` | Criar `MessageInput` bytes puros, chamar `carrinhoService.ProcessarMensagem` | Lógica de carrinho |
| `dto/carrinho.go` | Carrinho + Estados `aberto, aguardando_endereco_selecao, aguardando_endereco_novo` | Lógica |
| `dto/pedido.go` | `PedidoConfirmado`, `PedidoDTO`, `ItemPedidoDTO`, `CriarPedidoRequest` com `EnderecoEntrega` | - |
| `llm/provider.go` | UnifiedLLM - transcribe, keywords, resolve IDs via `textClient` + `formatCardapioForPrompt` | Salvar carrinho, Redis/MySQL |
| `service/carrinho_service.go` | Workflow + máquina de estados endereço + 1 SaveCarrinho + chama `clienteService` para endereço | Chamar repo direto |
| `service/pedido_service.go` | Só `pedidoRepo` + `cardapioService` - `ProcessarPedidoComEndereco(enderecoID *uint)` | Chamar `clienteRepo`/`enderecoRepo` |
| `service/cliente_service.go` | `AdicionarEndereco` (só Create), `ListarEnderecos`, `AtualizarUltimoPedido` | - |
| `helpers/formatter.go` | Mensagens WhatsApp com mesmo padrão visual `🛒 **SEU CARRINHO**` | Lógica de negócio |
| `server/http_server.go` | Subir Fiber, registrar rotas `/api` e `/webhook` | Chamar LLM, Redis cru |
| `server/api_handlers.go` | Dashboard - chama interfaces, nunca repo | - |

---

## 7. Wire - provider.go

```go
var providerSetHandlers = wire.NewSet(
  server.NewMCPServer,
  server.NewHttpServer,
  server.NewAPIHandlers,
  webhook.NewWhatsAppClient,
  webhook.NewProcessor,
  webhook.NewWebhookHandler, // só 1 vez - duplicado dava multiple bindings
)

var providerSetService = wire.NewSet(
  service.NewTenantService,
  service.NewAuthService,
  service.NewCardapioService,
  service.NewLLMService,
  service.NewPedidoService,
  service.NewClienteService,
  service.NewCarrinhoService, // agora recebe ClienteServiceInterface (6º param)
)
```

Erro comum: `providerSetHandlers has multiple bindings for *WebhookHandler` -> remove duplicata.

---

## 8. Git Workflow

Branch fixo: `main` (prod OCI) nunca quebra, `dev` integração. Branch temporário: `<tipo>/DEV-<num>-<kebab>` ex: `feature/DEV-8-cache-carrinho`. Commit: `feat(cache): adiciona GetOrSet - Refs #8`. Nunca `Closes #X`, kanban move manual. Fluxo: `dev` -> testa -> `main` -> `make build-push && make deploy`.

---

## 9. Changelog Recente

- **19/08/2026:** Migração `net/http` -> Fiber v2, remoção `sanitize.go` global, cache `GetOrSet` genérico, intent fast-path `bom dia` 0 token. -80% MySQL, 800->2500 req/s.
- **29/08/2026:** Fluxo endereço entrega - estados no carrinho Redis, `Pedido.EnderecoEntregaID`, `PedidoConfirmado.EnderecoEntrega *EnderecoDTO`, merge `pedido.go` + `pedido_dto.go` em um único arquivo, `formatter` com lista de endereços e confirmação com endereço, `pedido_service` corrigido sem chamar outros repos.

---

## 10. Como Rodar

```bash
cp .env.example .env
make init-db # rede mcp-network + MySQL/Redis
go mod download
make wire
go run ./cmd/server
curl http://localhost:8080/health
```

Teste endereço:
```
add item -> finalizar -> 0 endereços -> "Rua Teste, 123, Centro" -> pedido #X com EnderecoEntregaID
add item -> finalizar -> lista 1,2,3 -> "1" -> pedido com endereço 1
```

---

## 11. Próximos Passos

- [ ] ViaCEP para validar CEP no `parseEnderecoTexto`
- [ ] `IntentView` com `GetOrSet` carrinho
- [ ] `GenerateJWT` Fiber puro
- [ ] Teste E2E do fluxo endereço

---
Definido por Rafael Pasa - Pinhalzinho SC
