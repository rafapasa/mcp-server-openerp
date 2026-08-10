package main

import (
	"log"
	"os"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/database"
)

func main() {
	// Pega DSN das variáveis de ambiente
	cfg := config.LoadConfig()

	// Conexão com banco de dados
	db, err := database.NewMySQL(cfg, "")
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco: %v", err)
	}
	defer db.Close()

	// Roda migrações
	if err := database.Migrate(db.GetDB()); err != nil {
		log.Fatalf("Erro na migração: %v", err)
	}

	// (Opcional) Popula com dados iniciais
	if os.Getenv("SEED") == "true" {
		if err := database.Seed(db.GetDB()); err != nil {
			log.Fatalf("Erro no seed: %v", err)
		}
	}

	log.Println("✅ Migração finalizada!")
}
