-- 000006_create_payment_tables.up.sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `formas_pagamento` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `tenant_id` bigint unsigned NOT NULL,
  `nome` varchar(100) NOT NULL,
  `tipo` enum('dinheiro','pix','cartao_credito','cartao_debito') NOT NULL,
  `ativo` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_formas_pagamento_tenant_nome` (`tenant_id`, `nome`),
  KEY `idx_formas_pagamento_tenant_ativo` (`tenant_id`, `ativo`),
  CONSTRAINT `fk_formas_pagamento_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `pedido_pagamentos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `pedido_id` bigint unsigned NOT NULL,
  `forma_pagamento_id` bigint unsigned NOT NULL,
  `valor` decimal(10,2) NOT NULL,
  `troco_para` decimal(10,2) DEFAULT NULL,
  `status` enum('pendente','pago') NOT NULL DEFAULT 'pendente',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_pedido_pagamentos_pedido_id` (`pedido_id`),
  KEY `idx_pedido_pagamentos_forma_id` (`forma_pagamento_id`),
  CONSTRAINT `fk_pedido_pagamentos_pedido` FOREIGN KEY (`pedido_id`) REFERENCES `pedidos` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_pedido_pagamentos_forma` FOREIGN KEY (`forma_pagamento_id`) REFERENCES `formas_pagamento` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd
