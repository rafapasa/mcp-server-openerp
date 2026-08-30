# Trabalho: git e conduta

## Git
- **Commits: feitos SEMPRE pelo usuário.** O agente nunca roda `git commit` nem `git push`;
  entrega as mudanças no working tree (ou staged) para o usuário revisar e commitar.
- **Branch de trabalho padrão: `dev`** (integração). Branches específicas
  (`feature/DEV-<n>-*`) somente quando o usuário indicar. `main` é só produção/deploy.
- **Uso de git:** o agente só executa comandos git quando o usuário pedir explicitamente.
- **Verificação de branch:** ao iniciar cada tarefa, o agente verifica o branch atual
  e informa se não estiver em `dev` antes de começar qualquer trabalho.
- Branches: `main` (produção) · `dev` (integração) · `feature/DEV-<n>-<desc>`.
- Commit: `tipo: descrição — Refs #<n>` em pt-BR, uma mudança lógica por commit
  (`feat:`, `fix:`, `docs:`, ...).
- `make fmt` antes de commitar; não commitar `wire_gen.go` sem o `wire.go` atualizado.
- Deploy: `main ← dev` → `make build-push IMAGE_TAG=x.y.z` → `make deploy`.

## Conduta do agente
1. Responder e comentar em pt-BR.
2. Ler os arquivos envolvidos antes de alterar; seguir padrões existentes (nomenclatura, guard clauses, nulos).
3. Não afirmar fatos sem verificar no código; não assumir dependências fora do `go.mod`.
4. Código completo, sem placeholders nem `// ... resto do código`.
5. Validar ao final (build/test/lint) e confirmar.
6. Justificar tecnicamente ao terminar (performance, concorrência, design, trade-offs).
7. Mínima intervenção; refatorar só com justificativa + testes.
8. Mudança de comportamento/fluxo → atualizar `README.md`/`docs/`.
