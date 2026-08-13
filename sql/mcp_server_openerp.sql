-- ============================================
-- RESET DO BANCO DE DADOS
-- Remove todos os dados e recria a estrutura
-- ============================================

USE `mcp_server_openerp`;

-- ============================================
-- 1. REMOVE TODAS AS TABELAS (ORDEM CORRETA PARA FK)
-- ============================================
SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS `pedidos`;
DROP TABLE IF EXISTS `enderecos`;
DROP TABLE IF EXISTS `clientes`;
DROP TABLE IF EXISTS `produtos`;
DROP TABLE IF EXISTS `categorias`;
DROP TABLE IF EXISTS `tenants`;

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================
-- 2. RECRIA A ESTRUTURA
-- ============================================

-- 2.1 TENANTS
CREATE TABLE `tenants` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `nome` VARCHAR(100) NOT NULL,
    `cnpj` VARCHAR(18) DEFAULT NULL,
    `telefone` VARCHAR(20) DEFAULT NULL,
    `endereco` VARCHAR(255) DEFAULT NULL,
    `ativo` BOOLEAN DEFAULT TRUE,
    `created_at` DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2.2 CATEGORIAS
CREATE TABLE `categorias` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `nome` VARCHAR(50) NOT NULL,
    `descricao` VARCHAR(255) DEFAULT NULL,
    `ordem` BIGINT DEFAULT 0,
    `ativo` BOOLEAN DEFAULT TRUE,
    `created_at` DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_categoria_tenant` (`tenant_id`, `nome`),
    KEY `idx_categorias_tenant_id` (`tenant_id`),
    CONSTRAINT `fk_categorias_tenants` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2.3 PRODUTOS
CREATE TABLE `produtos` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `categoria_id` BIGINT UNSIGNED DEFAULT NULL,
    `nome` VARCHAR(100) NOT NULL,
    `codigo_barras` VARCHAR(50) DEFAULT NULL,
    `sku` VARCHAR(50) DEFAULT NULL,
    `descricao` VARCHAR(255) DEFAULT NULL,
    `preco` DECIMAL(10,2) NOT NULL,
    `unidade_medida` VARCHAR(20) DEFAULT 'un',
    `ingredientes` JSON DEFAULT NULL,
    `disponivel` BOOLEAN DEFAULT TRUE,
    `tempo_preparo` BIGINT DEFAULT 15,
    `created_at` DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    KEY `idx_produtos_tenant_id` (`tenant_id`),
    KEY `idx_produtos_categoria_id` (`categoria_id`),
    KEY `idx_produtos_tenant_disponivel` (`tenant_id`, `disponivel`),
    KEY `idx_produtos_nome` (`nome`),
    KEY `idx_produtos_tenant_categoria` (`tenant_id`, `categoria_id`),
    CONSTRAINT `fk_produtos_tenants` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_produtos_categorias` FOREIGN KEY (`categoria_id`) REFERENCES `categorias` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2.4 CLIENTES
CREATE TABLE `clientes` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `telefone` VARCHAR(20) NOT NULL COMMENT 'Número do WhatsApp',
    `nome` VARCHAR(100) DEFAULT NULL COMMENT 'Nome completo do cliente',
    `nome_perfil` VARCHAR(100) DEFAULT NULL COMMENT 'Nome do perfil do WhatsApp',
    `email` VARCHAR(100) DEFAULT NULL COMMENT 'E-mail do cliente',
    `inscricao_federal` VARCHAR(20) DEFAULT NULL COMMENT 'CPF (11 dígitos) ou CNPJ (14 dígitos)',
    `rg` VARCHAR(20) DEFAULT NULL COMMENT 'RG (apenas pessoa física)',
    `inscricao_estadual` VARCHAR(20) DEFAULT NULL COMMENT 'Inscrição Estadual (pessoa jurídica)',
    `inscricao_municipal` VARCHAR(20) DEFAULT NULL COMMENT 'Inscrição Municipal (pessoa jurídica)',
    `status` ENUM('ativo', 'inativo', 'pendente_validacao') NOT NULL DEFAULT 'ativo' COMMENT 'ativo, inativo, pendente_validacao',
    `status_reason` VARCHAR(255) DEFAULT NULL COMMENT 'Motivo do status (ex: mudança de dono)',
    `status_updated_at` TIMESTAMP NULL DEFAULT NULL COMMENT 'Data da última mudança de status',
    `nome_anterior` VARCHAR(100) DEFAULT NULL COMMENT 'Nome anterior (para validação)',
    `ultima_validacao_nome` TIMESTAMP NULL DEFAULT NULL COMMENT 'Data da última validação de nome',
    `ultimo_pedido_at` TIMESTAMP NULL DEFAULT NULL COMMENT 'Data do último pedido',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_cliente_tenant` (`tenant_id`),
    KEY `idx_cliente_telefone` (`telefone`),
    KEY `idx_cliente_status` (`status`),
    KEY `idx_cliente_ultimo_pedido` (`ultimo_pedido_at`),
    KEY `idx_cliente_created_at` (`created_at`),
    CONSTRAINT `fk_clientes_tenants` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Tabela de clientes do sistema';

-- 2.5 ENDEREÇOS
CREATE TABLE `enderecos` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `cliente_id` BIGINT UNSIGNED NOT NULL,
    `cep` VARCHAR(10) DEFAULT NULL COMMENT 'CEP (formato: 00000-000)',
    `logradouro` VARCHAR(255) NOT NULL COMMENT 'Nome da rua/avenida',
    `numero` VARCHAR(20) NOT NULL COMMENT 'Número do imóvel',
    `complemento` VARCHAR(100) DEFAULT NULL COMMENT 'Complemento (apto, bloco, etc)',
    `bairro` VARCHAR(100) DEFAULT NULL COMMENT 'Bairro',
    `cidade` VARCHAR(100) DEFAULT NULL COMMENT 'Cidade',
    `estado` VARCHAR(2) DEFAULT NULL COMMENT 'UF (SP, RJ, etc)',
    `pais` VARCHAR(50) DEFAULT 'Brasil' COMMENT 'País',
    `referencia` VARCHAR(255) DEFAULT NULL COMMENT 'Ponto de referência',
    `latitude` DECIMAL(10,8) DEFAULT NULL COMMENT 'Latitude (ex: -23.550520)',
    `longitude` DECIMAL(11,8) DEFAULT NULL COMMENT 'Longitude (ex: -46.633308)',
    `geolocalizacao_fonte` VARCHAR(50) DEFAULT NULL COMMENT 'Fonte: whatsapp, viacep, google, manual',
    `tipo` ENUM('residencial', 'comercial', 'entrega', 'cobranca') NOT NULL DEFAULT 'entrega',
    `principal` BOOLEAN DEFAULT FALSE COMMENT 'Endereço principal para entregas',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT 'Data de exclusão lógica (endereço inativo)',
    `observacoes` TEXT DEFAULT NULL COMMENT 'Observações sobre o endereço',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_endereco_cliente` (`cliente_id`),
    KEY `idx_endereco_cep` (`cep`),
    KEY `idx_endereco_principal` (`cliente_id`, `principal`),
    KEY `idx_endereco_tipo` (`tipo`),
    KEY `idx_endereco_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_enderecos_clientes` FOREIGN KEY (`cliente_id`) REFERENCES `clientes` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Endereços dos clientes (com soft delete)';

-- 2.6 PEDIDOS
CREATE TABLE `pedidos` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `cliente_id` BIGINT UNSIGNED DEFAULT NULL,
    `endereco_entrega_id` BIGINT UNSIGNED DEFAULT NULL,
    `cliente_nome` VARCHAR(100) DEFAULT NULL,
    `cliente_telefone` VARCHAR(20) DEFAULT NULL,
    `itens` JSON NOT NULL,
    `total` DECIMAL(10,2) NOT NULL,
    `status` ENUM('pendente', 'confirmado', 'preparando', 'entregue', 'cancelado') DEFAULT 'pendente',
    `observacoes` TEXT DEFAULT NULL,
    `tempo_estimado` BIGINT DEFAULT 0,
    `origem` ENUM('whatsapp', 'dashboard', 'api') DEFAULT 'whatsapp',
    `created_at` DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    KEY `idx_pedidos_tenant_id` (`tenant_id`),
    KEY `idx_pedidos_tenant_created` (`created_at`),
    KEY `idx_pedidos_tenant_status` (`tenant_id`, `status`, `created_at`),
    KEY `idx_pedidos_cliente` (`tenant_id`),
    KEY `idx_pedidos_status_created` (`status`, `created_at`),
    KEY `idx_pedidos_cliente_id` (`cliente_id`),
    KEY `idx_pedidos_endereco_entrega` (`endereco_entrega_id`),
    CONSTRAINT `fk_pedidos_tenants` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_pedidos_clientes` FOREIGN KEY (`cliente_id`) REFERENCES `clientes` (`id`) ON DELETE SET NULL,
    CONSTRAINT `fk_pedidos_enderecos` FOREIGN KEY (`endereco_entrega_id`) REFERENCES `enderecos` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- 3. VERIFICAÇÃO DA ESTRUTURA
-- ============================================
SHOW TABLES;

-- Verifica o número de tabelas
SELECT COUNT(*) as total_tables FROM information_schema.tables 
WHERE table_schema = 'mcp_server_openerp';
-- Deve retornar 6 (tenants, categorias, produtos, clientes, enderecos, pedidos)

-- Verifica as FK
SELECT 
    TABLE_NAME,
    CONSTRAINT_NAME,
    COLUMN_NAME,
    REFERENCED_TABLE_NAME,
    REFERENCED_COLUMN_NAME
FROM information_schema.KEY_COLUMN_USAGE
WHERE CONSTRAINT_SCHEMA = 'mcp_server_openerp'
AND REFERENCED_TABLE_NAME IS NOT NULL
ORDER BY TABLE_NAME, CONSTRAINT_NAME;

-- ============================================
-- 4. INSERTS DE TESTE (OPCIONAL - APENAS PARA VALIDAR)
-- ============================================

-- 4.1 Tenants
INSERT INTO `tenants` (`id`, `nome`, `cnpj`, `telefone`, `endereco`, `ativo`) VALUES 
(1, 'FastFood do Zé', '12.345.678/0001-90', '(11) 99999-9999', 'Rua das Flores, 123 - São Paulo/SP', 1),
(2, 'Mercado Popular', '23.456.789/0001-01', '(11) 88888-8888', 'Av. Principal, 456 - São Paulo/SP', 1),
(3, 'Farmácia Saúde', '34.567.890/0001-12', '(11) 77777-7777', 'Rua da Saúde, 789 - São Paulo/SP', 1);

-- 4.2 Clientes de Teste
INSERT INTO `clientes` (`tenant_id`, `telefone`, `nome`, `nome_perfil`, `email`, `status`) VALUES 
(1, '5511999999999', 'João Silva', 'João Silva', 'joao@email.com', 'ativo'),
(1, '5511988888888', 'Maria Santos', 'Maria S.', 'maria@email.com', 'ativo');

-- 4.3 Endereços de Teste
INSERT INTO `enderecos` (`cliente_id`, `cep`, `logradouro`, `numero`, `complemento`, `bairro`, `cidade`, `estado`, `latitude`, `longitude`, `principal`) VALUES 
(1, '01234-567', 'Rua das Flores', '123', 'Apto 42', 'Jardim Paulista', 'São Paulo', 'SP', -23.55052000, -46.63330800, 1),
(2, '04567-890', 'Rua dos Pinheiros', '456', NULL, 'Pinheiros', 'São Paulo', 'SP', -23.56240000, -46.69710000, 1);

-- 4.4 Categorias de Teste (FastFood)
INSERT INTO `categorias` (`tenant_id`, `nome`, `ordem`) VALUES 
(1, 'Lanches', 1),
(1, 'Bebidas', 2);

-- 4.5 Produtos de Teste
INSERT INTO `produtos` (`tenant_id`, `categoria_id`, `nome`, `descricao`, `preco`) VALUES 
(1, 1, 'X-Bacon', 'Pão, hambúrguer, bacon, queijo, alface, tomate', 29.90),
(1, 2, 'Coca-Cola Lata', 'Refrigerante Coca-Cola 350ml', 6.00);

-- 4.6 Pedido de Teste
INSERT INTO `pedidos` (`tenant_id`, `cliente_id`, `endereco_entrega_id`, `cliente_nome`, `cliente_telefone`, `itens`, `total`, `status`) VALUES 
(1, 1, 1, 'João Silva', '5511999999999', 
 '{"itens": [{"nome": "X-Bacon", "quantidade": 1, "preco": 29.90}]}', 
 29.90, 'confirmado');

-- Verifica dados inseridos
SELECT 'Tenants' as Tabela, COUNT(*) as Total FROM tenants UNION ALL
SELECT 'Categorias', COUNT(*) FROM categorias UNION ALL
SELECT 'Produtos', COUNT(*) FROM produtos UNION ALL
SELECT 'Clientes', COUNT(*) FROM clientes UNION ALL
SELECT 'Enderecos', COUNT(*) FROM enderecos UNION ALL
SELECT 'Pedidos', COUNT(*) FROM pedidos;