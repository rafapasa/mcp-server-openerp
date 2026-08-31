# mcp-server-openerp

Servidor de pedidos multi-setor via WhatsApp (Meta Cloud API) + API dashboard. Go 1.22+ Fiber v2, cache Redis real, multi-LLM (Groq Whisper áudio, Gemini Vision imagem/PDF, DeepSeek intent) e fluxo de endereço de entrega com máquina de estados no carrinho.

[![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8)](https://go.dev/)
[![Fiber v2](https://img.shields.io/badge/Fiber-v2-00ACD7)](https://gofiber.io/)
[![License MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 🚀 Stack

- **Framework:** Fiber v2 (fasthttp) - zero `net/http` em handlers
- **DI:** Google Wire (`make wire` gera `wire_gen.go`)
- **Cache:** Redis `GetOrSet[T]` genérico + `RedisInterface` (mockável) - elimina 80% MySQL
- **Banco:** MySQL 8.4 GORM + Soft Delete em endereços
- **LLMs:** Groq Whisper (áudio), Gemini Vision (imagem/PDF), DeepSeek (intent/itens)
- **Obs:** Zap logger, Prometheus, OTEL tracing, health checks Redis/MySQL

## 📁 Estrutura

```
cmd/
  server/main.go  # HTTP único Fiber
  stdio/main.go   # MCP stdio inspector
  migrate/main.go

internal/
  di/             # Wire
  dto/
    carrinho.go   # Carrinho com estados: aberto, aguardando_endereco_selecao, aguardando_endereco_novo
    pedido.go     # Merge único: PedidoDTO, ItemPedidoDTO, PedidoConfirmado com EnderecoEntrega
    cliente_dto.go
  helpers/
    formatter.go  # FormatResumoCarrinho, FormatRespostaPedido (com endereço), FormatListaEnderecos
  models/
    cliente.go    # Cliente + Endereco (Principal bool, DeletedAt)
    pedido.go     # Pedido.EnderecoEntregaID *uint + relation
  repository/     # cliente, endereco (Create/FindByClienteAtivos), pedido, produto
  service/
    carrinho_service.go # DONO WORKFLOW + máquina estados endereço
    pedido_service.go   # só pedidoRepo + cardapioService
    cliente_service.go  # AdicionarEndereco só Create
  llm/
    provider.go   # UnifiedLLM - dono fluxo LLM
    preprocessor.go # único lugar sanitiza
  webhook/
    whatsapp.go   # dedup 5min messageID + 200 rápido
    handlers.go   # MessageInput bytes puros
  server/
    http_server.go, api_handlers.go
```

## 🔄 Fluxo de Mensagem (29/08/2026 com Endereço)

```
WhatsApp POST /webhook
  -> dedup 5min + 200 <100ms
  -> MessageInput{Source, Text, Audio[], Image[]}
  -> carrinhoService.ProcessarMensagem (entry point único)
    ├─ Estado != aberto? -> handleSelecaoEndereco / handleNovoEndereco
    ├─ GetCarrinho Redis TTL 3600
    ├─ GetCardapio cache 1h
    ├─ intent.Classify fast-path 0 token
    ├─ LLM ObterTextoBase + Classificar + ResolverItens (prompt com contexto da loja: nome + segmento via GetPromptContext)
    ├─ mergeItem + 1 SaveCarrinho
    └─ finalizar -> iniciarFluxoFinalizacao:
         0 endereços -> FormatSolicitarNovoEndereco
         N endereços -> FormatListaEnderecos (1,2,3 + "novo")
         escolha -> FinalizarCarrinhoComEndereco(enderecoID) -> Limpar + AtualizarUltimoPedido
  -> SendMessage WhatsApp
```

Mídias: audio->Groq, image/PDF->Gemini base64, text direto.

## 📍 Regra Endereço (29/08/2026)

- **Imutável:** Só `Create`, nunca `Update`/`Delete`. Soft delete via `DeletedAt`.
- **Fluxo:** `finalizar` verifica endereços, lista ou pede novo `Rua, 123, Bairro, Cidade - UF, CEP`, parse por vírgula + regex CEP, cria e finaliza com `EnderecoEntregaID`.
- **Pedido:** `EnderecoEntregaID *uint` nullable (retirada = nil).

## ⚡ Quick Start

```bash
cp .env.example .env
make init-db
go mod download
make wire
go run ./cmd/server
curl http://localhost:8080/health
```

Teste:
```
quero um x-bacon -> finalizar -> "Rua Teste, 123, Centro" -> pedido #1 com endereço
quero coca -> finalizar -> "*1* - Rua Teste, 123" -> "1" -> pedido #2
```

## 🔧 Makefile

```bash
make dev              # air
make wire             # Wire gen
make build
make build-push IMAGE_TAG=1.0.0
make deploy
make test
make lint # go vet + gofmt -l
```

## 🔐 Env

| Var | Desc |
|-----|------|
| WEBHOOK_PORT | 8080 |
| WHATSAPP_VERIFY_TOKEN | Meta verify |
| WHATSAPP_TOKEN | Graph API |
| LLM_GROQ_KEY | Whisper |
| LLM_GEMINI_KEY | Vision |
| LLM_DEEPSEEK_KEY | Intent |
| REDIS_ADDR | :6379 |
| MYSQL_DSN | |

## 📦 Deploy OCI

```bash
git checkout main
git merge dev
make build-push
make deploy
./oci-mcp-server.sh
```

## 🧹 Breaking 19/08

- Removido `sanitize.go` global (quebrava preprocessor) -> sanitização só em `preprocessor.go`
- `net/http.ServeMux` -> Fiber `c.Body()`, `c.Query()`, `c.Locals("tenant_id")`

## 📚 Docs

- `docs/Contexto.md` - contexto único completo (este README + AGENTS + GIT_WORKFLOW + changelog + fluxo endereço)
- `docs/GIT_WORKFLOW.md` - branches `main/dev`, `feature/DEV-8-*`, commits `feat: ... Refs #8`
- `docs/AGENTS.md` - comandos AI agents

---
Rafael Pasa - Pinhalzinho SC
