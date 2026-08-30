# Engenharia: arquitetura, estilo e comandos

## Arquitetura
- `service/` = regra de negócio · `repository/` = acesso a dados (GORM) · `server/`+`webhook/` = entrega (Fiber).
- DTOs (`dto.`) entre camadas; `fiber.Ctx` só na entrega; service agnóstico a framework.
- DI: nunca instanciar structs dentro de handlers/serviços; construtor público retornando interface com deps por parâmetro:
  `NewCarrinhoService(...) CarrinhoServiceInterface`.
- Interfaces de serviço em `internal/service/interface.go`; repositórios declaram a própria (ex: `EnderecoRepositoryInterface`).
- Tocar DI → rodar `make wire`.

## Estilo Go
- `context.Context` primeiro parâmetro em toda operação I/O.
- Erros: wrap `%w` (`fmt.Errorf("...: %w", err)`); validar logo após cada chamada.
- Log: `logger.FromContext(ctx)` + `LogInfo/LogError` com campos tipados (`zap.Uint`, `zap.String`, `zap.Error`); nunca `fmt.Println`/`log.Printf`.
- Guard clauses; nomes sem redundância; comentário `// Nome explica o que faz`.
- LLMs/APIs externas: retry com backoff + validação de config prévia.
- Sem dependências novas sem necessidade real (conferir `go.mod` antes).

## Comandos
`make dev` (air) · `make wire` (DI) · `make build` · `make run-server` · `make run-stdio` ·
`make test` · `make lint` (golangci-lint) · `make fmt` (gofmt -w -s + goimports -w)

Checagens: `go build ./...` · `go vet ./...` · `gofmt -l .` (vazio) · `go test -v ./...`

**Concluir tarefa só após: `make fmt` + (`make wire` se DI) + `go build ./...` + testes.**
