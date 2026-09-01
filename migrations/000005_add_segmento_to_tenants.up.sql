-- 000005_add_segmento_to_tenants.up.sql
-- Campo segmento (farmacia, mercado, restaurante, geral) alinhado ao model Tenant.Segmento
-- +goose Up
-- +goose StatementBegin
ALTER TABLE `tenants`
ADD COLUMN `segmento` VARCHAR(50) NULL DEFAULT 'geral' AFTER `endereco`;
-- +goose StatementBegin
