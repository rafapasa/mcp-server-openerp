// internal/database/migrate.go
package database

import (
	"fmt"
	"log"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"gorm.io/gorm"
)

// Migrate executa todas as migrações
func Migrate(db *gorm.DB) error {
	log.Println("🚀 Iniciando migrações...")

	// 1. Migra as tabelas (ordem respeita dependências)
	if err := db.AutoMigrate(
		&models.Tenant{},
		&models.Categoria{},
		&models.Produto{},
		&models.Pedido{},
	); err != nil {
		return fmt.Errorf("erro ao migrar tabelas: %w", err)
	}

	// 2. Cria índices adicionais (GORM já cria os básicos)
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("erro ao criar índices: %w", err)
	}

	log.Println("✅ Migrações concluídas com sucesso!")
	return nil
}

// createIndexes cria índices adicionais para performance
func createIndexes(db *gorm.DB) error {
	queries := []string{
		// Índices compostos
		"CREATE INDEX idx_pedidos_tenant_status ON pedidos(tenant_id, status, created_at)",
		"CREATE INDEX idx_pedidos_tenant_created ON pedidos(tenant_id, created_at DESC)",
		"CREATE INDEX idx_produtos_tenant_disponivel ON produtos(tenant_id, disponivel)",
		"CREATE INDEX idx_pedidos_cliente ON pedidos(tenant_id, cliente_id)",

		// Índice único para categoria por tenant
		"CREATE UNIQUE INDEX uk_categoria_tenant ON categorias(tenant_id, nome)",
	}

	for _, query := range queries {
		// Ignora erro se o índice já existir
		if err := db.Exec(query).Error; err != nil {
			log.Printf("⚠️  Aviso ao criar índice: %v", err)
		}
	}

	return nil
}

// DropAllTables remove todas as tabelas (cuidado!)
func DropAllTables(db *gorm.DB) error {
	log.Println("⚠️  Removendo todas as tabelas...")

	if err := db.Migrator().DropTable(
		&models.Pedido{},
		&models.Produto{},
		&models.Categoria{},
		&models.Tenant{},
	); err != nil {
		return fmt.Errorf("erro ao dropar tabelas: %w", err)
	}

	log.Println("✅ Tabelas removidas!")
	return nil
}

// ResetDatabase reseta o banco (drop + migrate)
func ResetDatabase(db *gorm.DB) error {
	if err := DropAllTables(db); err != nil {
		return err
	}
	return Migrate(db)
}
