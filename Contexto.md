markdown

# 📄 Resumo da Jornada de Desenvolvimento - MCP Server para Pedidos Multi-Setor

**Data:** 10/08/2026  
**Status:** Em desenvolvimento ativo  
**Stack:** Go 1.23+, GORM, MySQL 8.4, Redis, MCP Server

---

## 🎯 Visão Geral do Projeto

Desenvolvimento de um servidor MCP (Model Context Protocol) em Go para processamento de pedidos via WhatsApp, atendendo múltiplos setores (restaurantes, mercados, farmácias). O sistema utiliza IA para interpretar mensagens e gerenciar carrinhos de compra.

---

## 🏗️ Arquitetura Final

### Estrutura de Diretórios

mcp-server-openerp/
├── cmd/
│ ├── stdio/main.go # Servidor STDIO
│ └── http/main.go # Servidor HTTP
├── internal/
│ ├── database/ # Migrações e conexões
│ ├── llm/ # Clientes LLM (OpenAI, Groq, Gemini)
│ │ ├── interface.go # Interface LLMClient
│ │ ├── factory.go # Fábrica de clientes
│ │ ├── helpers.go # Funções auxiliares
│ │ ├── openai.go
│ │ ├── groq.go
│ │ └── gemini.go
│ ├── models/ # Models GORM
│ ├── repository/ # Repositórios
│ ├── service/ # Serviços (Cardápio, Pedido, Carrinho)
│ │ ├── interfaces.go # Interfaces dos serviços
│ │ ├── cardapio_service.go
│ │ ├── pedido_service.go
│ │ └── carrinho_service.go
│ └── server/ # Servidor MCP
│ ├── server.go
│ ├── llm.go # Lógica de extração
│ ├── database.go
│ └── tools/ # Tools MCP
│ ├── registry.go
│ ├── whatsapp.go # Tool principal
│ ├── carrinho.go # Tools do carrinho
│ ├── pedido.go
│ ├── cardapio.go
│ └── helpers.go
└── .env
text


---

## 🔧 Principais Decisões Técnicas

| Decisão | Motivo |
|---------|--------|
| **Go + GORM + MySQL** | Equipe já tem expertise |
| **Redis para carrinho** | Performance e TTL automático |
| **Multi-LLM via interface** | Flexibilidade para trocar provedores |
| **Padrão Repository + Service** | Separação de responsabilidades |
| **Tools MCP** | Integração padronizada com IA |
| **Carrinho por cliente_id+tenant_id** | Isolamento entre estabelecimentos |

---

## 📊 Modelos de Dados (MySQL)

### Tabelas Principais

```sql
tenants          # Estabelecimentos (restaurante, mercado, farmácia)
categorias       # Categorias de produtos (FK tenant)
produtos         # Itens do cardápio (FK tenant, categoria)
pedidos          # Pedidos finalizados (FK tenant)

Campos JSON

    pedidos.itens: Array de itens do pedido

    produtos.ingredientes: Array de ingredientes

Estrutura SQL Completa
sql

-- ============================================
-- 1. TENANTS (Restaurantes/Mercados/Farmácias)
-- ============================================
CREATE TABLE tenants (
    id INT PRIMARY KEY AUTO_INCREMENT,
    nome VARCHAR(100) NOT NULL,
    cnpj VARCHAR(18),
    telefone VARCHAR(20),
    endereco VARCHAR(255),
    ativo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- 2. CATEGORIAS
-- ============================================
CREATE TABLE categorias (
    id INT PRIMARY KEY AUTO_INCREMENT,
    tenant_id INT NOT NULL,
    nome VARCHAR(50) NOT NULL,
    ordem INT DEFAULT 0,
    ativo BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE KEY uk_categoria_tenant (tenant_id, nome)
);

-- ============================================
-- 3. PRODUTOS (Cardápio)
-- ============================================
CREATE TABLE produtos (
    id INT PRIMARY KEY AUTO_INCREMENT,
    tenant_id INT NOT NULL,
    categoria_id INT,
    nome VARCHAR(100) NOT NULL,
    descricao VARCHAR(255),
    preco DECIMAL(10,2) NOT NULL,
    ingredientes JSON,
    disponivel BOOLEAN DEFAULT TRUE,
    tempo_preparo INT DEFAULT 15,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (categoria_id) REFERENCES categorias(id) ON DELETE SET NULL
);

-- ============================================
-- 4. PEDIDOS
-- ============================================
CREATE TABLE pedidos (
    id INT PRIMARY KEY AUTO_INCREMENT,
    tenant_id INT NOT NULL,
    cliente_id VARCHAR(50),
    cliente_nome VARCHAR(100),
    cliente_telefone VARCHAR(20),
    itens JSON NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    status ENUM('pendente','confirmado','preparando','entregue','cancelado') DEFAULT 'pendente',
    observacoes TEXT,
    tempo_estimado INT DEFAULT 0,
    origem ENUM('whatsapp','dashboard','api') DEFAULT 'whatsapp',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_tenant_created (tenant_id, created_at)
);

-- ============================================
-- 5. ÍNDICES ADICIONAIS
-- ============================================
CREATE INDEX idx_produtos_tenant ON produtos(tenant_id);
CREATE INDEX idx_produtos_disponivel ON produtos(tenant_id, disponivel);
CREATE INDEX idx_produtos_categoria ON produtos(categoria_id);
CREATE INDEX idx_pedidos_cliente ON pedidos(tenant_id, cliente_id);

🔄 Fluxo do WhatsApp
text

Mensagem → LLM (detecta intenção) → 
├─ "adicionar"   → Adicionar ao carrinho
├─ "remover"     → Remover do carrinho  
├─ "finalizar"   → Criar pedido no banco
├─ "limpar"      → Limpar carrinho
└─ "visualizar"  → Mostrar carrinho atual

Intenções Suportadas
Ação	Palavras-chave
adicionar	quero, adicionar, incluir, mais, +
remover	remover, tirar, excluir, -
finalizar	finalizar, confirmar, fechar, pagar
limpar	limpar, apagar, esvaziar
visualizar	(padrão) mostrar, ver, carrinho
🛠️ Tools MCP Disponíveis
Tool	Descrição
processar_mensagem_whatsapp	Principal - detecta intenção e processa
adicionar_ao_carrinho	Adiciona item específico
remover_do_carrinho	Remove item específico
visualizar_carrinho	Mostra carrinho atual
finalizar_pedido	Converte carrinho em pedido
limpar_carrinho	Limpa todo o carrinho
processar_pedido_restaurante	Dashboard manual
consultar_cardapio	Consulta cardápio
💾 Carrinho no Redis
Estrutura do Carrinho
json

{
  "cliente_id": "5511999999999",
  "tenant_id": "1",
  "itens": [
    {
      "nome": "X-Bacon",
      "quantidade": 2,
      "observacao": "sem cebola",
      "preco": 29.90
    }
  ],
  "created_at": "2026-08-10T10:00:00Z",
  "updated_at": "2026-08-10T10:05:00Z"
}

Configuração

    TTL: 1 hora (3600 segundos)

    Chave: carrinho:{tenant_id}:{cliente_id}

    Expiração automática para carrinhos abandonados

🤖 Multi-LLM Suportados
Provedores Disponíveis
Provedor	Variável	Modelo Padrão
OpenAI	LLM_PROVIDER=openai	gpt-4o-mini
Groq	LLM_PROVIDER=groq	llama-3.3-70b-versatile
Gemini	LLM_PROVIDER=gemini	gemini-2.0-flash
Configuração no .env
bash

# ============================================
# CONFIGURAÇÃO DO LLM
# ============================================

# Provedor: openai, groq, gemini
LLM_PROVIDER=groq

# Chave da API (use a chave do provedor escolhido)
LLM_API_KEY=sua_chave_api_aqui

# Modelo (opcional - usa o padrão se não especificado)
LLM_MODEL=llama-3.3-70b-versatile

# URL Base (opcional - usa o padrão se não especificado)
LLM_BASE_URL=

# ============================================
# PROVEDORES ESPECÍFICOS (opcionais)
# ============================================

# OpenAI
OPENAI_API_KEY=sua_chave_openai
OPENAI_MODEL=gpt-4o-mini

# Groq
GROQ_API_KEY=sua_chave_groq
GROQ_MODEL=llama-3.3-70b-versatile

# Gemini
GEMINI_API_KEY=sua_chave_gemini
GEMINI_MODEL=gemini-2.0-flash

# ============================================
# BANCO DE DADOS
# ============================================

DATABASE_DSN=root:root@tcp(localhost:3306)/mcp_server_openerp?charset=utf8mb4&parseTime=True&loc=Local

# ============================================
# REDIS
# ============================================

REDIS_ADDR=localhost:6379

📋 Interface do LLM
go

// internal/llm/interface.go

type LLMClient interface {
    // Generate envia um prompt e retorna a resposta
    Generate(prompt string) (string, error)
    
    // GenerateWithContext envia um prompt com contexto
    GenerateWithContext(ctx context.Context, prompt string) (string, error)
    
    // GetModel retorna o nome do modelo sendo usado
    GetModel() string
    
    // GetProvider retorna o nome do provedor
    GetProvider() string
    
    // ExtractIntent extrai a intenção do cliente da mensagem
    ExtractIntent(mensagem string, cardapio []ProdutoItem) (*IntencaoCliente, error)
}

type IntencaoCliente struct {
    Acao     string               // adicionar, remover, finalizar, visualizar, limpar
    Itens    []ItemPedidoInput
    Mensagem string
}

🔧 Interfaces dos Serviços
go

// internal/service/interfaces.go

type CardapioServiceInterface interface {
    GetCardapio(tenantID string) ([]ProdutoItem, error)
    ItemExisteNoCardapio(cardapio []ProdutoItem, nome string) (bool, float64)
    EncontrarItemSimilar(cardapio []ProdutoItem, nome string) string
    FormatarCardapio(cardapio []ProdutoItem) string
}

type PedidoServiceInterface interface {
    ProcessarPedido(tenantID, clienteID, clienteNome string, pedidoExtraido *PedidoExtraido) (*PedidoConfirmado, error)
}

type CarrinhoServiceInterface interface {
    AdicionarItem(clienteID, tenantID string, item ItemCarrinho) error
    RemoverItem(clienteID, tenantID string, nome string, quantidade int) error
    GetCarrinho(clienteID, tenantID string) (*Carrinho, error)
    LimparCarrinho(clienteID, tenantID string) error
    FinalizarCarrinho(clienteID, tenantID, clienteNome string) (*PedidoConfirmado, error)
    CalcularTotal(carrinho *Carrinho) float64
    CalcularTempoEstimado(carrinho *Carrinho) int
}

🧪 Como Testar
1. Configurar o Ambiente
bash

# Clone o repositório
git clone https://github.com/rafapasa/mcp-server-openerp.git
cd mcp-server-openerp

# Configure o .env
cp .env.example .env
# Edite o .env com suas chaves

# Instale as dependências
go mod download

2. Rodar o Servidor
bash

# Modo STDIO (para testes com MCP Inspector)
go run ./cmd/stdio/main.go

# Modo HTTP (para produção)
go run ./cmd/http/main.go

3. Testar com MCP Inspector
bash

# Instala o inspector
npm install -g @modelcontextprotocol/inspector

# Roda o inspector com seu servidor
npx @modelcontextprotocol/inspector -- go run ./cmd/stdio/main.go

# Acesse no navegador: http://localhost:6274

4. Exemplo de Mensagem
text

"quero um x-bacon e uma coca gelada"

5. Exemplo de Resposta Esperada
text

✅ Adicionado: 1x **X-Bacon** ao carrinho!
✅ Adicionado: 1x **Coca-Cola** ao carrinho!

🛒 **SEU CARRINHO**

• 1x **X-Bacon** - R$ 29.90
• 1x **Coca-Cola** - R$ 8.00
---
💰 **Total: R$ 37.90**
⏱️ **Tempo estimado:** 20 minutos

📝 *Comandos:*
• Adicionar: *quero mais um X-Bacon*
• Remover: *remover Coca-Cola*
• Finalizar: *finalizar pedido*
• Limpar: *limpar carrinho*

📁 Principais Arquivos para Revisão
Arquivo	Descrição
internal/llm/interface.go	Interface LLMClient e IntencaoCliente
internal/llm/helpers.go	Funções auxiliares (cleanJSON, extractJSON, validação)
internal/service/carrinho_service.go	Lógica do carrinho no Redis
internal/server/tools/whatsapp.go	Tool principal de WhatsApp
internal/server/tools/registry.go	Registro central de tools
internal/server/server.go	Servidor MCP principal
internal/models/models.go	Models GORM
internal/database/migrate.go	Migrações
🚀 Próximos Passos Sugeridos
Fase 1: Completar Implementação

    □

    Implementar ExtractIntent no GeminiLLM
    □

    Implementar ExtractIntent no GroqLLM
    □

    Implementar ExtractIntent no OpenAILLM

Fase 2: Testes e Qualidade

    □

    Adicionar testes unitários
    □

    Adicionar testes de integração
    □

    Configurar CI/CD

Fase 3: Infraestrutura

    □

    Implementar endpoint HTTP
    □

    Adicionar webhook para integração com ERP
    □

    Implementar autenticação (API keys)
    □

    Configurar logs estruturados
    □

    Adicionar métricas e monitoramento

Fase 4: Produção

    □

    Deploy em produção
    □

    Configurar backup do Redis
    □

    Implementar rate limiting
    □

    Adicionar health checks

✅ Status Atual
Componente	Status
Estrutura de banco de dados (MySQL)	✅ Concluído
Models GORM	✅ Concluído
Repositórios	✅ Concluído
Serviços (Cardápio, Pedido, Carrinho)	✅ Concluído
Tools MCP (WhatsApp, Carrinho, Pedido, Cardápio)	✅ Concluído
Interface LLM multi-provedor	✅ Concluído
Carrinho no Redis com TTL	✅ Concluído
Estrutura de diretórios organizada	✅ Concluído
Implementação do ExtractIntent	⏳ Em andamento
Testes unitários	⏳ Pendente
Endpoint HTTP	⏳ Pendente
📝 Notas de Desenvolvimento
1. Convenção de Nomes

    Constantes exportadas: CamelCase (ex: WavingHandIcon)

    Interfaces: NomeInterface (ex: LLMClient)

    Structs: NomeStruct (ex: CarrinhoService)

2. Padrões Utilizados

    Repository Pattern: Acesso a dados

    Service Pattern: Lógica de negócio

    Factory Pattern: Criação de clientes LLM

    Strategy Pattern: Múltiplos provedores LLM

3. Dependências Principais
go

require (
    github.com/go-sql-driver/mysql v1.7.0
    github.com/mark3labs/mcp-go v0.0.0-20240801012345-1234567890
    github.com/redis/go-redis/v9 v9.5.1
    gorm.io/driver/mysql v1.5.0
    gorm.io/gorm v1.25.0
)

4. Variáveis de Ambiente Obrigatórias

    LLM_PROVIDER (openai, groq, gemini)

    LLM_API_KEY (chave do provedor escolhido)

    DATABASE_DSN (conexão MySQL)

    REDIS_ADDR (endereço Redis)

Fim do Resumo