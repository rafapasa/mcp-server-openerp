-- 000003_create_produtos.up.sql
CREATE TABLE IF NOT EXISTS `produtos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `tenant_id` bigint unsigned NOT NULL,
  `categoria_id` bigint unsigned DEFAULT NULL,
  `nome` varchar(255) NOT NULL,
  `codigo_barras` varchar(50) DEFAULT NULL,
  `sku` varchar(50) DEFAULT NULL,
  `descricao` text,
  `preco` decimal(10,2) NOT NULL DEFAULT '0.00',
  `unidade_medida` varchar(20) DEFAULT 'un',
  `ingredientes` text,
  `disponivel` tinyint(1) DEFAULT '1',
  `tempo_preparo` int DEFAULT 5,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_produtos_tenant_id` (`tenant_id`),
  KEY `idx_produtos_categoria_id` (`categoria_id`),
  KEY `idx_produtos_nome` (`nome`),
  KEY `idx_produtos_sku` (`sku`),
  CONSTRAINT `fk_produtos_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_produtos_categoria` FOREIGN KEY (`categoria_id`) REFERENCES `categorias` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

