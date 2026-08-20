# Git Workflow - OpenERP

> Processo leve, seguro e escalável. Sem burocracia, com controle total.

## 1. Branches Fixos

| Branch | Propósito | Pode quebrar? | Deploy OCI? |
|--------|-----------|---------------|-------------|
| `main` | Produção - o que está rodando na OCI | NUNCA | Sim - `make build-push && make deploy` |
| `dev` | Desenvolvimento diário - integração | Pode | Não |

**Regra de ouro:** Nunca codar direto na `main`. A `main` só recebe `merge` quando `dev` está 100% testado.

## 2. Branches Temporários

### Formato
```
<tipo>/DEV-<numero>-<descricao-curta-kebab-case>
```

### Tipos

| Tipo | Quando usar | Exemplo |
|------|-------------|---------|
| `feature/` | Nova funcionalidade | `feature/DEV-8-cache-carrinho-intent-view` |
| `fix/` | Correção de bug | `fix/DEV-9-validar-hmac-whatsapp` |
| `client/` | Customização específica de cliente | `client/fastfood-pix-burguer-king` |
| `docs/` | Documentação | `docs/DEV-6-architecture-md` |
| `chore/` | Infra, deps, build | `chore/atualiza-go-1.22` |

**Por que usar esse padrão?**
- Em `git branch -a` você sabe o tipo, o ticket e o que faz em 1 segundo
- Link direto com GitHub Projects: `DEV-8` = Issue #8
- Onboarding de novos devs sem perguntas

## 3. Fluxo de Trabalho Diário

```bash
# 1. Começa o dia - atualiza dev
git checkout dev
git pull origin dev

# 2. Pega uma issue do board (ex: #8)
git checkout -b feature/DEV-8-cache-carrinho-intent-view

# 3. Trabalha e commita (Refs linka, não fecha auto)
git add .
git commit -m "feat(cache): adiciona GetOrSet no carrinho - Refs #8"

# 4. Termina feature - merge em dev
git checkout dev
git merge feature/DEV-8-cache-carrinho-intent-view
git branch -d feature/DEV-8-cache-carrinho-intent-view
git push origin dev

# 5. Testa dev com calma. Tudo 100%? Libera pra prod
git checkout main
git merge dev
git push origin main

# 6. Deploy OCI (sempre a partir da main)
make build-push
make deploy
```

## 4. Commits

### Formato
```
<tipo>(<escopo>): <mensagem curta>

<detalhes opcionais>

Refs #<numero>
```

### Tipos de commit

- `feat`: nova feature
- `fix`: correção de bug
- `docs`: documentação
- `chore`: infra, build, deps
- `refactor`: refatoração sem mudar comportamento
- `perf`: melhoria de performance

### Exemplos

```bash
# Boa - linka mas não fecha auto (você move kanban na mão)
git commit -m "feat(intent): classifier v2 fonético com levenshtein

- Add metaphone PT-BR para 'bon di' ~ 'bom dia'
- Add multi-intento GreetingWithAdd

Refs #10"

# WIP - trabalho em progresso
git commit -m "wip: iniciando cache carrinho - Refs #8"
```

**IMPORTANTE:** Não usar `Closes #X` / `Fixes #X` por enquanto. Fechamento de issues e movimento no kanban será MANUAL para manter acompanhamento e controle.

## 5. GitHub Projects (Kanban)

Board: https://github.com/users/rafapasa/projects/2/views/1

| Coluna | Quando mover |
|--------|--------------|
| `Backlog` | Issue criada |
| `Ready` | Pronta pra fazer |
| `In progress` | Começou a codar (move na mão ao criar branch) |
| `In review` | Merge em `dev` - testando |
| `Done` | Merge em `main` + deploy OCI OK |

Movimento é **manual** - sem automação por enquanto.

## 6. Customização por Cliente (Futuro)

Quando houver necessidade (5 clientes na fila: auto peças, fastfood, clínicas, labs):

```
/internal
  /core      -> regra igual para todos
  /clients
    /autopecas
    /fastfood
    /clinica
```

Branch `client/<nome>-<feature>` só mexe em `/clients/<nome>`. Nunca quebra core.

> **Não criar agora.** Criar quando a primeira customização real aparecer (YAGNI).

## 7. Comandos Úteis

```bash
# Ver branches
git branch -a

# Ver o que tem em dev que não tem em main (o que vai pra prod)
git log main..dev --oneline

# Desfazer último commit mantendo alterações (se commitou errado)
git reset --soft HEAD~1

# Ver diff entre dev e main
git diff main..dev

# Limpar branches locais já mergeados
git branch --merged | grep -v "main\|dev" | xargs git branch -d
```

---

**Definido em:** 20/08/2025  
**Por:** Rafael Pasa  
**Stack:** Go + Fiber + Redis + MySQL + OCI  
**Princípio:** Seguro, tranquilo, sem exageros. O que precisa quando for necessário.
