# AGENTS.md

Fonte canônica das regras de desenvolvimento: `.clinerules/` (lida automaticamente pelo Cline).
Este arquivo é um resumo para outras ferramentas/agentes.

## Quick Reference

| Task | Command |
|------|---------|
| Hot reload | `make dev` |
| DI (Wire) | `make wire` |
| Build | `make build` |
| Test | `make test` |
| Lint | `make lint` |
| Format | `make fmt` |
| Rodar HTTP | `make run-server` |
| Rodar MCP stdio | `make run-stdio` |

## Stack (resumo)

Go 1.25 · Fiber v2 (zero `net/http` em handlers) · GORM MySQL 8.4 · Redis (`GetOrSet[T]`) ·
Google Wire DI · mark3labs/mcp-go · zap (via `internal/observability/logger`).

## Regras críticas

- Todo fluxo WhatsApp entra por `carrinhoService.ProcessarMensagem` (entry point único).
- Endereço imutável: só `Create`; soft delete via `DeletedAt`.
- Sanitização apenas em `internal/llm/preprocessor.go`.
- DI: construtor público retornando interface; `context.Context` sempre primeiro parâmetro em I/O.

## Estrutura

`cmd/{server,stdio,migrate}` · `internal/{di,dto,helpers,models,repository,service,llm,webhook,server,observability,database,config,intent}`
