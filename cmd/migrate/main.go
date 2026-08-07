package main

import (
	"log"
	"os"

	"github.com/rafapasa/mcp-server-openerp/internal/database"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Pega DSN das variáveis de ambiente
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		// Exemplo: user:pass@tcp(localhost:3306)/mcp_fastfood?charset=utf8mb4&parseTime=True&loc=Local
		dsn = "root:root@tcp(localhost:3306)/mcp_fastfood?charset=utf8mb4&parseTime=True&loc=Local"
	}

	// Conecta ao banco
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Erro ao conectar: %v", err)
	}

	// Roda migrações
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Erro na migração: %v", err)
	}

	// (Opcional) Popula com dados iniciais
	if os.Getenv("SEED") == "true" {
		if err := database.Seed(db); err != nil {
			log.Fatalf("Erro no seed: %v", err)
		}
	}

	log.Println("✅ Migração finalizada!")
}
