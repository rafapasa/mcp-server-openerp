package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/rafapasa/mcp-server-openerp/migrations"
)

func main() {
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		// fallback para seu .env: usuario:senha@tcp(host:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "127.0.0.1"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "3306"
		}
		user := os.Getenv("DB_USER")
		dbpass := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		if user == "" {
			log.Fatal("DATABASE_DSN ou DB_USER/DB_PASSWORD/DB_NAME não definidos")
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, dbpass, host, port, dbname)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("erro ao conectar no MySQL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("erro ao pegar sql.DB: %v", err)
	}

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("mysql"); err != nil {
		log.Fatalf("dialect: %v", err)
	}

	dir := "."
	if len(os.Args) > 1 && os.Args[1] == "up" || len(os.Args) > 1 && os.Args[1] == "down" {
		// goose up / down
		cmd := os.Args[1]
		if cmd == "up" {
			if err := ensureWhatsAppSchema(sqlDB); err != nil {
				log.Fatalf("preparar schema do WhatsApp: %v", err)
			}
			if err := goose.Up(sqlDB, dir); err != nil {
				log.Fatalf("goose up failed: %v", err)
			}
			fmt.Println("✅ migrations up aplicadas")
		} else if cmd == "down" {
			if err := goose.Down(sqlDB, dir); err != nil {
				log.Fatalf("goose down failed: %v", err)
			}
			fmt.Println("✅ rollback aplicado")
		}
	} else {
		// default: up
		if err := ensureWhatsAppSchema(sqlDB); err != nil {
			log.Fatalf("preparar schema do WhatsApp: %v", err)
		}
		if err := goose.Up(sqlDB, dir); err != nil {
			log.Fatalf("goose up failed: %v", err)
		}
		fmt.Println("✅ migrations up aplicadas (default)")
	}
}

func ensureWhatsAppSchema(db *sql.DB) error {
	columns := []struct {
		name  string
		def   string
		after string
	}{
		{"waba_id", "VARCHAR(100) NULL", "endereco"},
		{"whatsapp_phone_id", "VARCHAR(100) NULL", "waba_id"},
		{"whatsapp_display_number", "VARCHAR(20) NULL", "whatsapp_phone_id"},
		{"whatsapp_verify_token", "VARCHAR(500) NULL", "whatsapp_display_number"},
		{"segmento", "VARCHAR(50) NULL DEFAULT 'geral'", "whatsapp_verify_token"},
	}
	for _, column := range columns {
		var count int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'tenants' AND column_name = ?",
			column.name,
		).Scan(&count); err != nil {
			return fmt.Errorf("verificar coluna %s: %w", column.name, err)
		}
		if count == 0 {
			query := fmt.Sprintf(
				"ALTER TABLE `tenants` ADD COLUMN `%s` %s AFTER `%s`",
				column.name, column.def, column.after,
			)
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("criar coluna %s: %w", column.name, err)
			}
		}
	}

	if _, err := db.Exec("UPDATE `tenants` SET `whatsapp_phone_id`='000000000000000', `whatsapp_display_number`='00000000000' WHERE `id`=2 AND `whatsapp_phone_id` IS NULL"); err != nil {
		return fmt.Errorf("preencher tenant 2: %w", err)
	}
	if _, err := db.Exec("UPDATE `tenants` SET `whatsapp_phone_id`='111111111111111', `whatsapp_display_number`='11111111111' WHERE `id`=3 AND `whatsapp_phone_id` IS NULL"); err != nil {
		return fmt.Errorf("preencher tenant 3: %w", err)
	}

	indexes := []struct {
		name string
		ddl  string
	}{
		{"uk_tenants_whatsapp_phone_id", "ALTER TABLE `tenants` ADD UNIQUE INDEX `uk_tenants_whatsapp_phone_id` (`whatsapp_phone_id`)"},
		{"idx_tenants_waba_id", "ALTER TABLE `tenants` ADD INDEX `idx_tenants_waba_id` (`waba_id`)"},
	}
	for _, index := range indexes {
		var count int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'tenants' AND index_name = ?",
			index.name,
		).Scan(&count); err != nil {
			return fmt.Errorf("verificar índice %s: %w", index.name, err)
		}
		if count == 0 {
			if _, err := db.Exec(index.ddl); err != nil {
				return fmt.Errorf("criar índice %s: %w", index.name, err)
			}
		}
	}

	productIndexes := []struct {
		name string
		ddl  string
	}{
		{"idx_produtos_tenant_disponivel", "ALTER TABLE `produtos` ADD INDEX `idx_produtos_tenant_disponivel` (`tenant_id`, `disponivel`)"},
		{"idx_produtos_tenant_nome", "ALTER TABLE `produtos` ADD INDEX `idx_produtos_tenant_nome` (`tenant_id`, `nome`)"},
	}
	for _, index := range productIndexes {
		var count int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'produtos' AND index_name = ?",
			index.name,
		).Scan(&count); err != nil {
			return fmt.Errorf("verificar índice %s: %w", index.name, err)
		}
		if count == 0 {
			if _, err := db.Exec(index.ddl); err != nil {
				return fmt.Errorf("criar índice %s: %w", index.name, err)
			}
		}
	}
	return nil
}
