-- 000004_add_whatsapp_fields_to_tenants.up.sql
-- Adiciona campos do WhatsApp e faz backfill seguro
-- +goose Up

-- +goose StatementBegin
ALTER TABLE `tenants`
ADD COLUMN `waba_id` VARCHAR(100) NULL AFTER `endereco`,
ADD COLUMN `whatsapp_phone_id` VARCHAR(100) NULL AFTER `waba_id`,
ADD COLUMN `whatsapp_display_number` VARCHAR(20) NULL AFTER `whatsapp_phone_id`,
ADD COLUMN `whatsapp_verify_token` VARCHAR(100) NULL AFTER `whatsapp_display_number`,
ADD COLUMN `segmento` VARCHAR(50) NULL DEFAULT 'geral' AFTER `whatsapp_verify_token`;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE `tenants` SET `whatsapp_phone_id`='000000000000000', `whatsapp_display_number`='00000000000' WHERE `id`=2 AND `whatsapp_phone_id` IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE `tenants` SET `whatsapp_phone_id`='111111111111111', `whatsapp_display_number`='11111111111' WHERE `id`=3 AND `whatsapp_phone_id` IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE `tenants` ADD UNIQUE INDEX `uk_tenants_whatsapp_phone_id` (`whatsapp_phone_id`);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE `tenants` ADD INDEX `idx_tenants_waba_id` (`waba_id`);
-- +goose StatementEnd