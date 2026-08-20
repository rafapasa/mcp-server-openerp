# mcp-server-openerp

Servidor de webhook WhatsApp (Meta) + API dashboard em Go 1.22+ com Fiber, cache real com Redis e roteamento inteligente de mídia (texto, áudio, imagem, PDF).

[![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8)](https://go.dev/)
[![Fiber v2](https://img.shields.io/badge/Fiber-v2-00ACD7)](https://gofiber.io/)
[![License MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 🚀 Stack

- **Framework:** Fiber v2 (fasthttp) - sem `net/http`, sem `ServeMux`
- **Cache:** Redis + `GetOrSet` genérico (elimina 80% de MySQL)
- **LLMs:** 
  - Groq Whisper (transcrição áudio)
  - Gemini Vision (imagem/receita)
  - DeepSeek (extração intent)
- **Banco:** MySQL (GORM) + Redis
- **Obs:** Zap logger, Prometheus metrics, OTEL tracing

## 📁 Estrutura

```
cmd/
├── api/          # Dashboard API
└── webhook/      # Webhook WhatsApp

internal/
├── api/          # Handlers Fiber (Dashboard, Pedidos, Clientes, Produtos)
├── webhook/      # Handlers Fiber (WhatsApp webhook + mídia)
├── cache/        # GetOrSet genérico + InvalidateByTenant
├── intent/       # Classify rápido (greeting, smalltalk) - 0 token
├── llm/
│   ├── preprocessor.go  # Limpeza cirúrgica (2 etapas)
│   ├── deepseek.go
│   └── gemini.go
├── media/        # Groq transcriber
├── observability/ # logger, metrics, tracing, middleware (rate limit, security headers)
├── repository/   # GORM repos
├── service/     # Cardapio, Pedido, Cliente, Carrinho
└── dto/
```

## 🔄 Fluxo de Mensagem

```
WhatsApp -> Fiber POST /webhook (raw, sem sanitize global)
   ↓
1. intent.Classify(raw) -> "bom dia"? -> GreetingResponse (5ms, 0 token, 0 Redis)
   ↓
2. preprocessor.Process(raw) -> Cleaned + Medidas + Numeros
   ↓
3. cache.GetOrSet("cardapio:{tenant}", 5m) -> MySQL só se miss
   ↓
4. LLM ExtractIntent(Cleaned + candidatos) -> adicionar/remover/finalizar
   ↓
5. FormatarResumoCarrinho / FinalizarCarrinho
   ↓
WhatsApp SendMessage
```

**Mídias:**
- `audio/voice` -> DownloadMedia -> Groq Whisper -> texto
- `image` -> DownloadMedia -> base64 -> Gemini Vision -> JSON receita/pedido
- `document/pdf` -> Gemini Vision -> texto

## ⚡ Quick Start

```bash
cp .env.example .env
# configure WHATSAPP_VERIFY_TOKEN, DB, REDIS, LLM keys

make init-db          # cria rede docker mcp-network + sobe MySQL/Redis
go mod download
go run ./cmd/webhook  # :8080
go run ./cmd/api      # :8081

curl http://localhost:8080/health
curl http://localhost:8081/health
```

## 🔧 Makefile

```bash
make dev              # air live reload
make build            # bin/api bin/webhook
make build-push IMAGE_TAG=1.0.0
make deploy IMAGE_TAG=1.0.0
make test
make lint
```

## 🔐 Env

| Var | Desc |
|-----|------|
| `WEBHOOK_PORT` | 8080 |
| `API_PORT` | 8081 |
| `WHATSAPP_VERIFY_TOKEN` | token Meta webhook |
| `WHATSAPP_TOKEN` | token Graph API |
| `LLM_PROVIDER` | deepseek |
| `LLM_GROQ_MODEL` | whisper-large-v3 |
| `LLM_GEMINI_MODEL` | gemini-2.0-flash |
| `LLM_DEEPSEEK_MODEL` | deepseek-chat |
| `REDIS_ADDR` | |
| `MYSQL_DSN` | |

## 🧹 O que foi removido

- `internal/observability/middleware/sanitize.go` - sanitização global quebrava `preprocessor.go`. Agora sanitização só no preprocessor (2 etapas).
- `http.NewServeMux` - trocado por Fiber em `api/server.go` e `webhook/routes.go`
- `net/http` handlers - tudo `func(c *fiber.Ctx) error`

## 📦 Deploy

```bash
make init-db
make build-push IMAGE_TAG=1.0.0
make deploy IMAGE_TAG=1.0.0
```
