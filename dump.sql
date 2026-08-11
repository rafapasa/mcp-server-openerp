CREATE DATABASE  IF NOT EXISTS `mcp_server_openerp` /*!40100 DEFAULT CHARACTER SET utf8mb3 */ /*!80016 DEFAULT ENCRYPTION='N' */;
USE `mcp_server_openerp`;
-- MySQL dump 10.13  Distrib 8.0.46, for Win64 (x86_64)
--
-- Host: localhost    Database: mcp_server_openerp
-- ------------------------------------------------------
-- Server version	8.4.4

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `categorias`
--

DROP TABLE IF EXISTS `categorias`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `categorias` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `tenant_id` bigint unsigned NOT NULL,
  `nome` varchar(50) NOT NULL,
  `descricao` varchar(255) DEFAULT NULL,
  `ordem` bigint DEFAULT '0',
  `ativo` tinyint(1) DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_categoria_tenant` (`tenant_id`,`nome`),
  KEY `idx_categorias_tenant_id` (`tenant_id`),
  CONSTRAINT `fk_tenants_categorias` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `categorias`
--

LOCK TABLES `categorias` WRITE;
/*!40000 ALTER TABLE `categorias` DISABLE KEYS */;
INSERT INTO `categorias` VALUES (1,1,'Lanches',NULL,1,1,'2026-08-10 17:21:18.000'),(2,1,'Bebidas',NULL,2,1,'2026-08-10 17:21:18.000'),(3,1,'Acompanhamentos',NULL,3,1,'2026-08-10 17:21:18.000'),(4,1,'Sobremesas',NULL,4,1,'2026-08-10 17:21:18.000'),(5,2,'Hortifruti',NULL,1,1,'2026-08-10 17:21:32.000'),(6,2,'Padaria',NULL,2,1,'2026-08-10 17:21:32.000'),(7,2,'Laticínios',NULL,3,1,'2026-08-10 17:21:32.000'),(8,2,'Bebidas',NULL,4,1,'2026-08-10 17:21:32.000'),(9,3,'Medicamentos',NULL,1,1,'2026-08-10 17:21:42.000'),(10,3,'Cosméticos',NULL,2,1,'2026-08-10 17:21:42.000'),(11,3,'Higiene Pessoal',NULL,3,1,'2026-08-10 17:21:42.000'),(12,3,'Suplementos',NULL,4,1,'2026-08-10 17:21:42.000');
/*!40000 ALTER TABLE `categorias` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pedidos`
--

DROP TABLE IF EXISTS `pedidos`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pedidos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `tenant_id` bigint unsigned NOT NULL,
  `cliente_id` varchar(50) DEFAULT NULL,
  `cliente_nome` varchar(100) DEFAULT NULL,
  `cliente_telefone` varchar(20) DEFAULT NULL,
  `itens` json NOT NULL,
  `total` decimal(10,2) NOT NULL,
  `status` enum('pendente','confirmado','preparando','entregue','cancelado') DEFAULT 'pendente',
  `observacoes` text,
  `tempo_estimado` bigint DEFAULT '0',
  `origem` enum('whatsapp','dashboard','api') DEFAULT 'whatsapp',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_pedidos_tenant_id` (`tenant_id`),
  KEY `idx_pedidos_cliente_id` (`cliente_id`),
  KEY `idx_pedidos_tenant_created` (`created_at`),
  KEY `idx_pedidos_tenant_status` (`tenant_id`,`status`,`created_at`),
  KEY `idx_pedidos_cliente` (`tenant_id`,`cliente_id`),
  KEY `idx_pedidos_status_created` (`status`,`created_at`),
  CONSTRAINT `fk_tenants_pedidos` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pedidos`
--

LOCK TABLES `pedidos` WRITE;
/*!40000 ALTER TABLE `pedidos` DISABLE KEYS */;
INSERT INTO `pedidos` VALUES (1,1,'65996908844 ','Rafael pasa','65996908844 ','[{\"nome\": \"X-Bacon\", \"observacao\": \"\", \"quantidade\": 1, \"preco_unitario\": 29.9}, {\"nome\": \"Água Mineral\", \"observacao\": \"bem gelada\", \"quantidade\": 1, \"preco_unitario\": 3.5}, {\"nome\": \"Coca-Cola Lata\", \"observacao\": \"bem gelada\", \"quantidade\": 1, \"preco_unitario\": 6}]',39.40,'confirmado','Bebidas bem geladas',30,'whatsapp','2026-08-10 18:29:20.464','2026-08-10 18:29:20.464'),(2,3,'65996908844','Rafael Pasa','65996908844','[{\"nome\": \"Máscara de Cílios\", \"observacao\": \"\", \"quantidade\": 1, \"preco_unitario\": 35.9}, {\"nome\": \"Losartana 50mg\", \"observacao\": \"\", \"quantidade\": 1, \"preco_unitario\": 32.9}]',68.80,'confirmado','Cliente solicitou informação de preço',25,'whatsapp','2026-08-10 18:34:41.142','2026-08-10 18:34:41.142'),(3,3,'65996908844','Rafael Pasa','65996908844','[{\"nome\": \"Paracetamol 750mg\", \"observacao\": \"Solicitado Tylenol; prefere genérico se disponível\", \"quantidade\": 1, \"preco_unitario\": 12.9}, {\"nome\": \"Dipirona 500mg\", \"observacao\": \"Solicitado 1mg; prefere genérico se disponível\", \"quantidade\": 1, \"preco_unitario\": 8.9}]',21.80,'confirmado','Cliente aceita medicamentos genéricos se houver opção',25,'whatsapp','2026-08-10 18:40:51.612','2026-08-10 18:40:51.612'),(4,1,'65996908844','Olise','65996908844','[{\"nome\": \"X-Bacon\", \"observacao\": \"\", \"quantidade\": 1, \"preco_unitario\": 29.9}, {\"nome\": \"Coca-Cola Lata\", \"observacao\": \"zero\", \"quantidade\": 1, \"preco_unitario\": 6}]',35.90,'confirmado','',25,'whatsapp','2026-08-10 18:44:26.611','2026-08-10 18:44:26.611');
/*!40000 ALTER TABLE `pedidos` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `produtos`
--

DROP TABLE IF EXISTS `produtos`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `produtos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `tenant_id` bigint unsigned NOT NULL,
  `categoria_id` bigint unsigned DEFAULT NULL,
  `nome` varchar(100) NOT NULL,
  `codigo_barras` varchar(50) DEFAULT NULL,
  `sku` varchar(50) DEFAULT NULL,
  `descricao` varchar(255) DEFAULT NULL,
  `preco` decimal(10,2) NOT NULL,
  `unidade_medida` varchar(20) DEFAULT 'un',
  `ingredientes` json DEFAULT NULL,
  `disponivel` tinyint(1) DEFAULT '1',
  `tempo_preparo` bigint DEFAULT '15',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_produtos_tenant_id` (`tenant_id`),
  KEY `idx_produtos_categoria_id` (`categoria_id`),
  KEY `idx_produtos_tenant_disponivel` (`tenant_id`,`disponivel`),
  KEY `idx_produtos_nome` (`nome`),
  KEY `idx_produtos_tenant_categoria` (`tenant_id`,`categoria_id`),
  CONSTRAINT `fk_categorias_produtos` FOREIGN KEY (`categoria_id`) REFERENCES `categorias` (`id`),
  CONSTRAINT `fk_tenants_produtos` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=97 DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `produtos`
--

LOCK TABLES `produtos` WRITE;
/*!40000 ALTER TABLE `produtos` DISABLE KEYS */;
INSERT INTO `produtos` VALUES (1,1,1,'X-Bacon',NULL,NULL,'Pão, hambúrguer, bacon, queijo, alface, tomate',29.90,'un','[\"pão\", \"hambúrguer\", \"bacon\", \"queijo\", \"alface\", \"tomate\"]',1,15,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(2,1,1,'X-Tudo',NULL,NULL,'Pão, hambúrguer, bacon, ovo, queijo, alface, tomate',34.90,'un','[\"pão\", \"hambúrguer\", \"bacon\", \"ovo\", \"queijo\", \"alface\", \"tomate\"]',1,18,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(3,1,1,'Hambúrguer Simples',NULL,NULL,'Pão, hambúrguer, queijo',19.90,'un','[\"pão\", \"hambúrguer\", \"queijo\"]',1,10,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(4,1,1,'X-Salada',NULL,NULL,'Pão, hambúrguer, queijo, alface, tomate, maionese',24.90,'un','[\"pão\", \"hambúrguer\", \"queijo\", \"alface\", \"tomate\", \"maionese\"]',1,12,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(5,1,1,'X-Egg',NULL,NULL,'Pão, hambúrguer, ovo, queijo, alface, tomate',26.90,'un','[\"pão\", \"hambúrguer\", \"ovo\", \"queijo\", \"alface\", \"tomate\"]',1,13,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(6,1,1,'X-Frango',NULL,NULL,'Pão, filé de frango, queijo, alface, tomate, maionese',27.90,'un','[\"pão\", \"filé de frango\", \"queijo\", \"alface\", \"tomate\", \"maionese\"]',1,14,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(7,1,1,'X-Bacon Duplo',NULL,NULL,'Pão, 2 hambúrgueres, bacon, queijo, alface, tomate',36.90,'un','[\"pão\", \"2 hambúrgueres\", \"bacon\", \"queijo\", \"alface\", \"tomate\"]',1,20,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(8,1,1,'Veggie Burger',NULL,NULL,'Pão, hambúrguer de grão-de-bico, queijo, alface, tomate',31.90,'un','[\"pão\", \"hambúrguer de grão-de-bico\", \"queijo\", \"alface\", \"tomate\"]',1,15,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(9,1,2,'Coca-Cola 2L',NULL,NULL,'Refrigerante Coca-Cola 2 litros',12.00,'un','[\"água\", \"açúcar\", \"cafeína\"]',1,0,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(10,1,2,'Coca-Cola Lata',NULL,NULL,'Refrigerante Coca-Cola 350ml',6.00,'un','[\"água\", \"açúcar\", \"cafeína\"]',1,0,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(11,1,2,'Guaraná 2L',NULL,NULL,'Refrigerante Guaraná 2 litros',10.00,'un','[\"água\", \"açúcar\", \"guaraná\"]',1,0,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(12,1,2,'Suco de Laranja',NULL,NULL,'Suco natural de laranja 500ml',8.00,'un','[\"laranja\", \"água\", \"açúcar\"]',1,0,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(13,1,2,'Suco de Uva',NULL,NULL,'Suco natural de uva 500ml',9.00,'un','[\"uva\", \"água\", \"açúcar\"]',1,0,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(14,1,2,'Água Mineral',NULL,NULL,'Água mineral 500ml',3.50,'un','[\"água\", \"minerais\"]',1,0,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(15,1,2,'Água com Gás',NULL,NULL,'Água com gás 500ml',4.50,'un','[\"água\", \"gás carbônico\"]',1,0,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(16,1,2,'Milkshake Chocolate',NULL,NULL,'Milkshake de chocolate 500ml',15.00,'un','[\"leite\", \"sorvete de chocolate\", \"calda de chocolate\"]',1,5,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(17,1,3,'Batata Frita',NULL,NULL,'Batata frita crocante porção média',14.00,'un','[\"batata\", \"sal\", \"óleo\"]',1,10,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(18,1,3,'Batata Frita Cheddar',NULL,NULL,'Batata frita com cheddar e bacon',18.00,'un','[\"batata\", \"cheddar\", \"bacon\", \"sal\"]',1,12,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(19,1,3,'Onion Rings',NULL,NULL,'Anéis de cebola empanados',12.00,'un','[\"cebola\", \"farinha\", \"óleo\", \"sal\"]',1,10,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(20,1,3,'Mandioca Frita',NULL,NULL,'Mandioca frita crocante porção média',13.00,'un','[\"mandioca\", \"sal\", \"óleo\"]',1,10,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(21,1,3,'Polenta Frita',NULL,NULL,'Polenta frita crocante porção média',11.00,'un','[\"polenta\", \"sal\", \"óleo\"]',1,8,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(22,1,3,'Salada Simples',NULL,NULL,'Salada de alface, tomate e cebola',8.00,'un','[\"alface\", \"tomate\", \"cebola\"]',1,5,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(23,1,3,'Batata Rústica',NULL,NULL,'Batata rústica com alecrim',15.00,'un','[\"batata\", \"alecrim\", \"sal\", \"azeite\"]',1,12,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(24,1,3,'Nuggets de Frango',NULL,NULL,'Nuggets de frango empanados (6 unidades)',16.00,'un','[\"frango\", \"farinha\", \"sal\"]',1,8,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(25,1,4,'Milkshake Chocolate',NULL,NULL,'Milkshake de chocolate 500ml',15.00,'un','[\"leite\", \"sorvete de chocolate\", \"calda\"]',1,5,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(26,1,4,'Milkshake Morango',NULL,NULL,'Milkshake de morango 500ml',15.00,'un','[\"leite\", \"sorvete de morango\", \"calda de morango\"]',1,5,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(27,1,4,'Milkshake Baunilha',NULL,NULL,'Milkshake de baunilha 500ml',15.00,'un','[\"leite\", \"sorvete de baunilha\", \"calda\"]',1,5,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(28,1,4,'Sundae Chocolate',NULL,NULL,'Sundae de chocolate com calda e granulado',12.00,'un','[\"sorvete de chocolate\", \"calda de chocolate\", \"granulado\"]',1,3,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(29,1,4,'Sundae Morango',NULL,NULL,'Sundae de morango com calda e granulado',12.00,'un','[\"sorvete de morango\", \"calda de morango\", \"granulado\"]',1,3,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(30,1,4,'Petit Gâteau',NULL,NULL,'Petit Gâteau com calda de chocolate e sorvete',18.00,'un','[\"chocolate\", \"farinha\", \"ovos\", \"sorvete\"]',1,15,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(31,1,4,'Pudim',NULL,NULL,'Pudim de leite condensado com calda de caramelo',10.00,'un','[\"leite condensado\", \"ovos\", \"leite\", \"açúcar\"]',1,0,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(32,1,4,'Torta de Limão',NULL,NULL,'Torta de limão com merengue',12.00,'un','[\"limão\", \"leite condensado\", \"biscoito\", \"merengue\"]',1,0,'2026-08-10 17:21:18.000','2026-08-10 17:21:18.000'),(33,2,5,'Maçã',NULL,NULL,'Maçã Gala - kg',8.90,'un','[\"maçã\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(34,2,5,'Banana',NULL,NULL,'Banana Prata - kg',5.90,'un','[\"banana\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(35,2,5,'Alface',NULL,NULL,'Alface Crespa - unidade',3.50,'un','[\"alface\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(36,2,5,'Tomate',NULL,NULL,'Tomate Italiano - kg',7.90,'un','[\"tomate\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(37,2,5,'Cebola',NULL,NULL,'Cebola - kg',4.90,'un','[\"cebola\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(38,2,5,'Batata',NULL,NULL,'Batata Inglesa - kg',5.90,'un','[\"batata\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(39,2,5,'Laranja',NULL,NULL,'Laranja Pêra - kg',4.50,'un','[\"laranja\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(40,2,5,'Abacaxi',NULL,NULL,'Abacaxi - unidade',6.90,'un','[\"abacaxi\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(41,2,6,'Pão Francês',NULL,NULL,'Pão francês - unidade',0.80,'un','[\"farinha\", \"fermento\", \"sal\", \"água\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(42,2,6,'Pão de Forma',NULL,NULL,'Pão de forma integral - 500g',8.90,'un','[\"farinha integral\", \"fermento\", \"sal\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(43,2,6,'Bolo Simples',NULL,NULL,'Bolo simples de baunilha',15.90,'un','[\"farinha\", \"ovos\", \"açúcar\", \"baunilha\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(44,2,6,'Torta Salgada',NULL,NULL,'Torta salgada de frango',22.90,'un','[\"massa\", \"frango\", \"milho\", \"ervilha\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(45,2,6,'Pão Integral',NULL,NULL,'Pão integral - 500g',10.90,'un','[\"farinha integral\", \"fermento\", \"sal\", \"grãos\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(46,2,6,'Biscoito Salgado',NULL,NULL,'Biscoito salgado - pacote 200g',4.50,'un','[\"farinha\", \"sal\", \"gordura\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(47,2,6,'Sonho',NULL,NULL,'Sonho com doce de leite',6.90,'un','[\"massa\", \"doce de leite\", \"açúcar\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(48,2,6,'Pão de Queijo',NULL,NULL,'Pão de queijo congelado - 500g',12.90,'un','[\"polvilho\", \"queijo\", \"ovos\", \"leite\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(49,2,7,'Leite Integral',NULL,NULL,'Leite integral - 1L',4.90,'un','[\"leite\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(50,2,7,'Queijo Mussarela',NULL,NULL,'Queijo mussarela - 500g',18.90,'un','[\"leite\", \"coalho\", \"sal\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(51,2,7,'Queijo Prato',NULL,NULL,'Queijo prato - 500g',19.90,'un','[\"leite\", \"coalho\", \"sal\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(52,2,7,'Iogurte Natural',NULL,NULL,'Iogurte natural - 200ml',3.90,'un','[\"leite\", \"fermento\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(53,2,7,'Requeijão',NULL,NULL,'Requeijão cremoso - 200g',7.90,'un','[\"leite\", \"creme\", \"sal\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(54,2,7,'Manteiga',NULL,NULL,'Manteiga com sal - 200g',8.90,'un','[\"creme de leite\", \"sal\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(55,2,7,'Creme de Leite',NULL,NULL,'Creme de leite - 200ml',4.50,'un','[\"leite\", \"gordura\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(56,2,7,'Leite Condensado',NULL,NULL,'Leite condensado - 395g',6.90,'un','[\"leite\", \"açúcar\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(57,2,8,'Refrigerante Coca 2L',NULL,NULL,'Coca-Cola 2L',10.90,'un','[\"água\", \"açúcar\", \"cafeína\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(58,2,8,'Suco de Laranja',NULL,NULL,'Suco de laranja - 1L',6.90,'un','[\"laranja\", \"água\", \"açúcar\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(59,2,8,'Água Mineral',NULL,NULL,'Água mineral - 1.5L',2.90,'un','[\"água\", \"minerais\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(60,2,8,'Cerveja',NULL,NULL,'Cerveja Pilsen - 350ml',3.90,'un','[\"cevada\", \"lúpulo\", \"água\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(61,2,8,'Vinho Tinto',NULL,NULL,'Vinho tinto seco - 750ml',25.90,'un','[\"uva\", \"sulfito\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(62,2,8,'Energético',NULL,NULL,'Energético - 250ml',7.90,'un','[\"taurina\", \"cafeína\", \"açúcar\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(63,2,8,'Chá Gelado',NULL,NULL,'Chá gelado - 1.5L',4.50,'un','[\"chá\", \"água\", \"açúcar\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(64,2,8,'Cerveja IPA',NULL,NULL,'Cerveja IPA - 350ml',5.90,'un','[\"cevada\", \"lúpulo\", \"água\"]',1,0,'2026-08-10 17:21:32.000','2026-08-10 17:21:32.000'),(65,3,9,'Dipirona 500mg',NULL,NULL,'Dipirona sódica 500mg - caixa 10 comprimidos',8.90,'un','[\"dipirona\", \"lactose\", \"amido\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(66,3,9,'Paracetamol 750mg',NULL,NULL,'Paracetamol 750mg - caixa 10 comprimidos',12.90,'un','[\"paracetamol\", \"amido\", \"estearato\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(67,3,9,'Amoxicilina 500mg',NULL,NULL,'Amoxicilina 500mg - caixa 10 cápsulas',25.90,'un','[\"amoxicilina\", \"lactose\", \"amido\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(68,3,9,'Omeprazol 20mg',NULL,NULL,'Omeprazol 20mg - caixa 7 cápsulas',18.90,'un','[\"omeprazol\", \"lactose\", \"amido\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(69,3,9,'Losartana 50mg',NULL,NULL,'Losartana 50mg - caixa 30 comprimidos',32.90,'un','[\"losartana\", \"lactose\", \"amido\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(70,3,9,'Azitromicina 500mg',NULL,NULL,'Azitromicina 500mg - caixa 3 comprimidos',35.90,'un','[\"azitromicina\", \"amido\", \"estearato\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(71,3,9,'Clonazepam 2mg',NULL,NULL,'Clonazepam 2mg - caixa 30 comprimidos',28.90,'un','[\"clonazepam\", \"lactose\", \"amido\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(72,3,9,'Fluoxetina 20mg',NULL,NULL,'Fluoxetina 20mg - caixa 30 comprimidos',42.90,'un','[\"fluoxetina\", \"lactose\", \"amido\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(73,3,10,'Base Líquida',NULL,NULL,'Base líquida média cobertura - 30ml',45.90,'un','[\"água\", \"silicone\", \"pigmento\", \"filtro solar\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(74,3,10,'Delineador',NULL,NULL,'Delineador líquido preto',22.90,'un','[\"água\", \"pigmento\", \"polímero\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(75,3,10,'Máscara de Cílios',NULL,NULL,'Máscara de cílios volume extra - 10ml',35.90,'un','[\"cera\", \"pigmento\", \"polímero\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(76,3,10,'Batom Matte',NULL,NULL,'Batom matte vermelho - 3.5g',29.90,'un','[\"cera\", \"óleo\", \"pigmento\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(77,3,10,'Blush',NULL,NULL,'Blush rosa - 5g',32.90,'un','[\"talco\", \"pigmento\", \"óleo\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(78,3,10,'Corretivo',NULL,NULL,'Corretivo líquido - 6ml',26.90,'un','[\"água\", \"pigmento\", \"silicone\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(79,3,10,'Pó Compacto',NULL,NULL,'Pó compacto matte - 10g',39.90,'un','[\"talco\", \"pigmento\", \"silicone\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(80,3,10,'Paleta de Sombras',NULL,NULL,'Paleta de sombras 12 cores',59.90,'un','[\"mica\", \"pigmento\", \"óleo\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(81,3,11,'Shampoo',NULL,NULL,'Shampoo nutritivo 350ml',25.90,'un','[\"água\", \"sulfato\", \"extratos vegetais\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(82,3,11,'Condicionador',NULL,NULL,'Condicionador hidratante 350ml',28.90,'un','[\"água\", \"silicone\", \"extratos vegetais\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(83,3,11,'Sabonete Líquido',NULL,NULL,'Sabonete líquido 250ml',15.90,'un','[\"água\", \"glicerina\", \"fragrância\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(84,3,11,'Creme Dental',NULL,NULL,'Creme dental 90g',6.90,'un','[\"flúor\", \"sílica\", \"menta\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(85,3,11,'Desodorante',NULL,NULL,'Desodorante aerosol 100ml',12.90,'un','[\"propelente\", \"alumínio\", \"fragrância\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(86,3,11,'Creme Hidratante',NULL,NULL,'Creme hidratante corporal 200ml',32.90,'un','[\"água\", \"glicerina\", \"manteiga\", \"fragrância\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(87,3,11,'Fio Dental',NULL,NULL,'Fio dental - pacote 50m',8.90,'un','[\"nylon\", \"cera\", \"menta\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(88,3,11,'Enxaguante Bucal',NULL,NULL,'Enxaguante bucal 500ml',14.90,'un','[\"álcool\", \"flúor\", \"menta\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(89,3,12,'Whey Protein',NULL,NULL,'Whey protein concentrado - 900g',89.90,'un','[\"proteína do soro do leite\", \"aminoácidos\", \"aroma\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(90,3,12,'Creatina',NULL,NULL,'Creatina monohidratada - 300g',59.90,'un','[\"creatina\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(91,3,12,'BCAA',NULL,NULL,'BCAA 2:1:1 - 200g',49.90,'un','[\"leucina\", \"isoleucina\", \"valina\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(92,3,12,'Vitamina C',NULL,NULL,'Vitamina C efervescente - 10 comprimidos',18.90,'un','[\"ácido ascórbico\", \"bicarbonato\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(93,3,12,'Ômega 3',NULL,NULL,'Ômega 3 - 60 cápsulas',45.90,'un','[\"óleo de peixe\", \"vitamina E\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(94,3,12,'Multivitamínico',NULL,NULL,'Multivitamínico - 30 comprimidos',35.90,'un','[\"vitaminas\", \"minerais\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(95,3,12,'Glucosamina',NULL,NULL,'Glucosamina - 60 cápsulas',55.90,'un','[\"glucosamina\", \"sulfato\", \"MSM\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000'),(96,3,12,'Colágeno',NULL,NULL,'Colágeno hidrolisado - 300g',69.90,'un','[\"colágeno\", \"vitamina C\"]',1,0,'2026-08-10 17:21:42.000','2026-08-10 17:21:42.000');
/*!40000 ALTER TABLE `produtos` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `tenants`
--

DROP TABLE IF EXISTS `tenants`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tenants` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `nome` varchar(100) NOT NULL,
  `cnpj` varchar(18) DEFAULT NULL,
  `telefone` varchar(20) DEFAULT NULL,
  `endereco` varchar(255) DEFAULT NULL,
  `ativo` tinyint(1) DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb3;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `tenants`
--

LOCK TABLES `tenants` WRITE;
/*!40000 ALTER TABLE `tenants` DISABLE KEYS */;
INSERT INTO `tenants` VALUES (1,'FastFood do Zé','12.345.678/0001-90','(11) 99999-9999','Rua das Flores, 123 - São Paulo/SP',1,'2026-08-10 17:21:18.000'),(2,'Mercado Popular','23.456.789/0001-01','(11) 88888-8888','Av. Principal, 456 - São Paulo/SP',1,'2026-08-10 17:21:32.000'),(3,'Farmácia Saúde','34.567.890/0001-12','(11) 77777-7777','Rua da Saúde, 789 - São Paulo/SP',1,'2026-08-10 17:21:42.000');
/*!40000 ALTER TABLE `tenants` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-08-10 22:42:39
