# Outros - Arquivo temporário de anotações

## Modelos Gemini (referência)

Gemini 3.0 Flash - gemini-3.0-flash
Gemini 2.5 Flash - gemini-2.5-flash
Gemini 2.5 Pro - gemini-2.5-pro
Gemini 2.0 Flash - gemini-2.0-flash
Gemini 2.0 Flash-Lite - gemini-2.0-flash-lite
Gemini 1.5 Pro - gemini-1.5-pro
Gemini 1.5 Flash - gemini-1.5-flash

## Template original MCP Go Starter - Features (não é deste projeto)

Este trecho veio do template mcp-go-starter e não pertence ao mcp-server-openerp, movido pra cá:

| Category | Feature | Description |
|----------|---------|-------------|
| **Tools** | `hello` | Basic tool with annotations |
| | `get_weather` | Tool returning structured data |
| | `ask_llm` | Tool that invokes LLM sampling |
| | `long_task` | Tool with 5-second progress updates |
| | `load_bonus_tool` | Dynamically loads a new tool |
| **Resources** | `info://about` | Static informational resource |
| | `file://example.md` | File-based markdown resource |
| **Templates** | `greeting://{name}` | Personalized greeting |
| | `data://items/{id}` | Data lookup by ID |
| **Prompts** | `greet` | Greeting in various styles |
| | `code_review` | Code review with focus areas |

### Exemplo Tool com Annotations (template)
```go
mcp.AddTool(server, &mcp.Tool{
    Name: "hello",
    Annotations: &mcp.ToolAnnotations{ReadOnlyHint: ptr(true)},
}, helloHandler)
```

## Build deploy - notas antigas

# Se tentar deploy sem db
make deploy IMAGE_TAG=1.0.0
# ❌ ERRO: rede mcp-network não existe! Rode primeiro: make init-db

# Correto
make init-db
make build-push IMAGE_TAG=1.0.0
make deploy IMAGE_TAG=1.0.0
