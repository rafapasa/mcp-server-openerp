-- 000007_add_pedido_status_delivery.up.sql
-- O ENUM de `pedidos.status` (novos valores `em_preparo` e `saiu_para_entrega`)
-- é preparado pelo migrador Go (ensurePedidoStatusSchema) antes da execução do Goose.
-- +goose Up
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
