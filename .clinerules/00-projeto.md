# Projeto: mcp-server-openerp

Pedidos multi-setor via WhatsApp (Meta Cloud API) + API dashboard.
Go 1.25 · Fiber v2 · GORM MySQL 8.4 · Redis · Google Wire · mark3labs/mcp-go.

## Stack (não reinventar)
- Fiber v2 (fasthttp): zero `net/http` em handlers.
- DI: Wire — `make wire` regenera `internal/di/wire_gen.go`.
- Cache: Redis `GetOrSet[T]` + `RedisInterface` (mockável) em `internal/database`.
- Banco: MySQL 8.4 + GORM; soft delete em endereços (`DeletedAt`).
- LLMs: Groq Whisper (áudio) · Gemini Vision (imagem/PDF) · DeepSeek (intent/itens) · Qwen local.
- Obs: zap via `internal/observability/logger` · Prometheus · OTEL · health checks.

## Estrutura (orientação)
`cmd/{server,stdio,migrate}` (entry points) →
`internal/{di,config,database,dto,helpers,models,repository,service,llm,webhook,server,observability,intent}`
- `service/carrinho_service.go` = dono do workflow · `llm/preprocessor.go` = único sanitize
- `helpers/` = formatters de resposta · `webhook/` = dedup + handlers · `server/` = Fiber

## Invariantes (nunca quebrar)
- Todo fluxo WhatsApp passa por `carrinhoService.ProcessarMensagem` — sem entry points paralelos.
- Endereço imutável: só `Create`; soft delete via `DeletedAt`; listar com `FindByClienteAtivos`.
- Carrinho: estados `aberto`, `aguardando_endereco_selecao`, `aguardando_endereco_novo` — respeitar máquina de estados.
- Pedido: `EnderecoEntregaID *uint` nullable (retirada = nil).
- Sanitização SÓ em `internal/llm/preprocessor.go` (sem sanitize global).
- Webhook: dedup 5min por messageID + 200 < 100ms; processamento assíncrono quando necessário.
