# Contexto - OpenERP mcp-server-openerp - Chat Fechamento

Data: 2026-09-04
Objetivo: Fechar chat gigante e preservar contexto para próximas issues

## 1. Projeto
- Repo: rafapasa/mcp-server-openerp
- Stack: Go, Fiber, GORM, MySQL, Redis (GetOrSet, SetNXWithContext), WhatsApp Cloud API
- Multi-tenant: tenants.id = 2 (Mercado), 3 (Farmácia) - teste com phone_number_id 000... e 111...

## 2. Problema Core Resolvido Neste Chat
- `processor.go` recebia `metadata.display_phone_number` e `phone_number_id` do webhook, mas `resolveTenant` usava só `GetByTelefone(display)` -> quebra com DDD, vazamento entre tenants
- Solução implementada: `resolveTenant(phoneNumberID, displayPhone)` tenta primeiro `GetByWhatsAppPhoneID` (com cache Redis `tenant:phone:{id}` 1h) e fallback para `GetByTelefone`

## 3. Arquivos Analisados / Fix Gerados (em /mnt/data/fixed/)
- models/tenant.go -> add WabaID, WhatsappPhoneID (unique), WhatsappDisplayNumber, WhatsappVerifyToken
- repository/tenant_repo.go -> novo método FindByWhatsAppPhoneID
- dto/tenant.go -> novos campos DTO
- service/tenant_service.go -> NewTenantService agora recebe (repo, cache) e método GetByWhatsAppPhoneID com GetOrSet
- webhook/processor.go -> resolveTenant com 2 params, debounce, logs
- migrations: técnica segura add NULL -> backfill -> UNIQUE INDEX
  - 000001_create_tenants (DDL real sem segmento)
  - 000002_categorias, 000003_produtos, 000004_add_whatsapp_fields (corrigida AFTER endereco, não AFTER segmento)

DDL real do banco:
```sql
CREATE TABLE `tenants` (
  `id` bigint unsigned AUTO_INCREMENT,
  `nome` varchar(100) NOT NULL,
  `cnpj` varchar(18),
  `telefone` varchar(20),
  `endereco` varchar(255),
  `ativo` tinyint(1) DEFAULT '1',
  `created_at` datetime(3),
  PRIMARY KEY (`id`)
)
```
Obs: models/tenant.go que me mandou tinha `segmento`, mas DDL não tem. Migration corrigida usa AFTER endereco.

## 4. Fluxo WhatsApp Analisado (prints image_2f5140, be862d, f4d855)
- Bom: LLM identifica "1 x bacon e um tudo" -> X-Bacon Especial + X-Tudo, incrementa coca com "coloca mais uma coca"
- Ruim: carrinho verboso repete bloco Comandos toda vez, flood 3 mensagens seguidas
- Ruim: endereço aceito sem ViaCEP, sem normalização (Ana albrecht)
- Falta: forma de pagamento, handoff humano, consulta pedido, notificações

## 5. Labels GitHub (image_eb4025.png)
21 labels ativos. Crítica: categoriza por tecnologia (go, fiber, cache, intent, llm) e não por valor. Sugestão manter só bug, enhancement, bot, webhook, observability, security, documentation, good first issue. Arquivar go, dependencies, github_actions, fiber, cache, intent.

## 6. Issues Planejadas - Script gerado em /mnt/data/create_issues.sh
Usa gh CLI, labels enxutos (enhancement,bot,webhook)

Grupo 1 - UX Core:
1. feat(bot): botões interativos WhatsApp (SendButtons + debounce 2s) - SEPARADA
2. refactor(bot): carrinho resumido (formatarCarrinhoResumido, sinônimos remover/tira)
3. feat(bot): validação ViaCEP + normalização

Grupo 2 - Conversão:
4. feat(bot): forma pagamento (dinheiro/troco, pix chave, cartão) - criar coluna forma_pagamento, troco_para em pedidos
5. feat(bot): falar com atendente handoff (chave Redis atendimento:humano:{clienteID} 30min, timeout 10min)

Grupo 3 - Pós-pedido:
6. feat(bot): consultar último pedido (intent cadê meu pedido, limite último pedido, privacidade por telefone)
7. feat(bot): notificação saiu para entrega (status enum, endpoint /pedidos/{id}/status, idempotência SetNX notificacao:saiu:{pedidoID})
8. feat(admin): notificação novo pedido para responsável configurável por tenant (tabela tenant_notificacoes_config: canal whatsapp/email/webhook, destino, eventos, ativo) - worker assíncrono com retry, HMAC webhook

## 7. Migrations - Pasta na raiz
Decisão: pasta `migrations/` na raiz (padrão goose/migrate). Estrutura gerada em /mnt/data/mcp-server-openerp/migrations/ com 000001 a 000004 + cmd/migrate/main.go com go:embed ../../migrations/*.sql

Uso:
```
go run ./cmd/migrate up
goose -dir ./migrations mysql "user:pass@tcp(127.0.0.1:3306)/openerp" up
```

## 8. Embedded Signup (discussão painel vs API)
Pergunta: precisa acessar developers.facebook.com para adicionar fones e pegar ids?
Resposta: Não, via Embedded Signup + WhatsApp Business Management API. Fluxo: botão "Conectar WhatsApp" (FB.login com scopes business_management, whatsapp_business_management) -> callback code -> exchange token -> GET /me/whatsapp_business_accounts com phone_numbers -> INSERT tenants (waba_id, whatsapp_phone_id, display). Para adicionar número novo em WABA existente: POST /{WABA_ID}/phone_numbers + POST /{phone_number_id}/register com pin SMS. Precisa Business Verification e permissão whatsapp_business_management aprovada.

## 9. Próximos Passos Acordados
- Usuário vai criar issues via script create_issues.sh
- Vai planejar Roadmap no GitHub Projects (views Backlog, Ready, In progress, In review, Done - image_d2ad83.png)
- Usar Roadmap view + Milestones v0.2 para lotes de alteração / versões
- Para cada issue, gerar contexto em arquivo docs/issues/{id}-{slug}.md e fechar chat, abrir novo para evitar chat gigante
- Não gerar código agora

## 10. Arquivos para levar para próximo chat
- /mnt/data/fixed/* (5 arquivos corrigidos)
- /mnt/data/mcp-server-openerp/migrations/*
- /mnt/data/create_issues.sh
- Este contexto

## 11. Comandos úteis
```bash
gh issue create --repo rafapasa/mcp-server-openerp --title "..." --label "enhancement,bot" --body "..."
```

Fim do contexto.
