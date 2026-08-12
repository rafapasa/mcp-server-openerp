-- ============================================
-- SCHEMA: mcp_server_openerp
-- Tabelas: clientes, enderecos
-- ============================================

USE `mcp_server_openerp`;

-- ============================================
-- 1. CLIENTES
-- ============================================
DROP TABLE IF EXISTS `clientes`;

CREATE TABLE `clientes` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    
    -- Identificação
    `telefone` VARCHAR(20) NOT NULL COMMENT 'Número do WhatsApp',
    `nome` VARCHAR(100) NULL COMMENT 'Nome completo do cliente',
    `nome_perfil` VARCHAR(100) NULL COMMENT 'Nome do perfil do WhatsApp',
    `email` VARCHAR(100) NULL COMMENT 'E-mail do cliente',
    `inscricao_federal` VARCHAR(20) NULL COMMENT 'CPF (11 dígitos) ou CNPJ (14 dígitos)',
    `rg` VARCHAR(20) NULL COMMENT 'RG (apenas pessoa física)',
    `inscricao_estadual` VARCHAR(20) NULL COMMENT 'Inscrição Estadual (pessoa jurídica)',
    `inscricao_municipal` VARCHAR(20) NULL COMMENT 'Inscrição Municipal (pessoa jurídica)',
    
    -- Status e Validação
    `status` ENUM('ativo', 'inativo', 'pendente_validacao') NOT NULL DEFAULT 'ativo' 
        COMMENT 'ativo, inativo, pendente_validacao',
    `status_reason` VARCHAR(255) NULL COMMENT 'Motivo do status (ex: mudança de dono)',
    `status_updated_at` TIMESTAMP NULL COMMENT 'Data da última mudança de status',
    
    -- Validação de Nome (mudança de dono)
    `nome_anterior` VARCHAR(100) NULL COMMENT 'Nome anterior (para validação)',
    `ultima_validacao_nome` TIMESTAMP NULL COMMENT 'Data da última validação de nome',
    
    -- Controle
    `ultimo_pedido_at` TIMESTAMP NULL COMMENT 'Data do último pedido',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    PRIMARY KEY (`id`),
    FOREIGN KEY (`tenant_id`) REFERENCES `tenants`(`id`) ON DELETE CASCADE,
    
    -- Índices
    INDEX `idx_cliente_tenant` (`tenant_id`),
    INDEX `idx_cliente_telefone` (`telefone`),
    INDEX `idx_cliente_status` (`status`),
    INDEX `idx_cliente_ultimo_pedido` (`ultimo_pedido_at`),
    INDEX `idx_cliente_created_at` (`created_at`)
    
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci 
COMMENT='Tabela de clientes do sistema';


-- ============================================
-- 2. ENDEREÇOS (COM SOFT DELETE)
-- ============================================
DROP TABLE IF EXISTS `enderecos`;

CREATE TABLE `enderecos` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `cliente_id` BIGINT UNSIGNED NOT NULL,
    
    -- Endereço
    `cep` VARCHAR(10) NULL COMMENT 'CEP (formato: 00000-000)',
    `logradouro` VARCHAR(255) NOT NULL COMMENT 'Nome da rua/avenida',
    `numero` VARCHAR(20) NOT NULL COMMENT 'Número do imóvel',
    `complemento` VARCHAR(100) NULL COMMENT 'Complemento (apto, bloco, etc)',
    `bairro` VARCHAR(100) NULL COMMENT 'Bairro',
    `cidade` VARCHAR(100) NULL COMMENT 'Cidade',
    `estado` VARCHAR(2) NULL COMMENT 'UF (SP, RJ, etc)',
    `pais` VARCHAR(50) DEFAULT 'Brasil' COMMENT 'País',
    
    -- Referência
    `referencia` VARCHAR(255) NULL COMMENT 'Ponto de referência',
    
    -- 📍 GEOLOCALIZAÇÃO
    `latitude` DECIMAL(10, 8) NULL COMMENT 'Latitude (ex: -23.550520)',
    `longitude` DECIMAL(11, 8) NULL COMMENT 'Longitude (ex: -46.633308)',
    `geolocalizacao_fonte` VARCHAR(50) NULL COMMENT 'Fonte: whatsapp, viacep, google, manual',
    
    -- Tipo e Padrão
    `tipo` ENUM('residencial', 'comercial', 'entrega', 'cobranca') NOT NULL DEFAULT 'entrega',
    `principal` BOOLEAN DEFAULT FALSE COMMENT 'Endereço principal para entregas',
    
    -- ⚠️ Soft Delete (IMPORTANTE)
    `deleted_at` TIMESTAMP NULL COMMENT 'Data de exclusão lógica (endereço inativo)',
    
    -- Metadados
    `observacoes` TEXT NULL COMMENT 'Observações sobre o endereço',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    PRIMARY KEY (`id`),
    FOREIGN KEY (`cliente_id`) REFERENCES `clientes`(`id`) ON DELETE CASCADE,
    
    -- Índices
    INDEX `idx_endereco_cliente` (`cliente_id`),
    INDEX `idx_endereco_cep` (`cep`),
    INDEX `idx_endereco_principal` (`cliente_id`, `principal`),
    INDEX `idx_endereco_tipo` (`tipo`),
    INDEX `idx_endereco_deleted_at` (`deleted_at`)  -- Para filtrar endereços ativos
    
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci 
COMMENT='Endereços dos clientes (com soft delete)';

-- -- ============================================
-- -- 3. AJUSTES NA TABELA PEDIDOS
-- -- ============================================
-- alter table `pedidos` drop column `client_id`;
-- -- Adicionar coluna cliente_id se não existir
-- ALTER TABLE `pedidos` 
--     ADD COLUMN `cliente_id` BIGINT UNSIGNED NULL AFTER `tenant_id`,
--     ADD FOREIGN KEY (`cliente_id`) REFERENCES `clientes`(`id`) ON DELETE SET NULL,
--     ADD INDEX `idx_pedidos_cliente_id` (`cliente_id`);

-- Adicionar coluna endereco_entrega_id (para referência)
ALTER TABLE `pedidos` 
    -- ADD COLUMN `endereco_entrega_id` BIGINT UNSIGNED NULL AFTER `cliente_id`,
    ADD FOREIGN KEY (`endereco_entrega_id`) REFERENCES `enderecos`(`id`) ON DELETE SET NULL,
    ADD INDEX `idx_pedidos_endereco_entrega` (`endereco_entrega_id`);


-- ============================================
-- 4. INSERTS PARA TESTE
-- ============================================
-- Clientes de exemplo
INSERT INTO `clientes` (
    `tenant_id`, `telefone`, `nome`, `nome_perfil`, `email`, 
    `inscricao_federal`, `status`, `ultimo_pedido_at`, `created_at`
) VALUES 
(1, '5511999999999', 'João Silva', 'João Silva', 'joao@email.com', '00000000191', 'ativo', NOW(), NOW()),
(1, '5511988888888', 'Maria Santos', 'Maria S.', 'maria@email.com', '00000000292', 'ativo', NOW(), NOW()),
(2, '5511977777777', 'Carlos Souza', 'Carlos Souza', 'carlos@email.com', '00000000393', 'ativo', NOW(), NOW());

-- Endereços de exemplo
INSERT INTO `enderecos` (
    `cliente_id`, `cep`, `logradouro`, `numero`, `complemento`, 
    `bairro`, `cidade`, `estado`, `principal`, `tipo`, 
    `latitude`, `longitude`, `geolocalizacao_fonte`
) VALUES 
(1, '01234-567', 'Rua das Flores', '123', 'Apto 42', 'Jardim Paulista', 'São Paulo', 'SP', TRUE, 'residencial', -23.550520, -46.633308, 'manual'),
(1, '01234-568', 'Av. Paulista', '1000', 'Conjunto 51', 'Cerqueira César', 'São Paulo', 'SP', FALSE, 'comercial', -23.561800, -46.655800, 'manual'),
(2, '04567-890', 'Rua dos Pinheiros', '456', NULL, 'Pinheiros', 'São Paulo', 'SP', TRUE, 'residencial', -23.562400, -46.697100, 'manual');