-- dev-11 não precisa migration de banco, cache é em Redis.
-- Mas se quiser garantir índice para performance de FindByTenantDisponiveis:

-- 000005_add_produtos_indexes.up.sql
-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_produtos_tenant_disponivel ON produtos(tenant_id, disponivel);
CREATE INDEX IF NOT EXISTS idx_produtos_tenant_nome ON produtos(tenant_id, nome);
-- +goose StatementEnd
