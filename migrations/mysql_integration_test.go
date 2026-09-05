package migrations

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
)

func TestMySQLMigrationsApply(t *testing.T) {
	dsn := os.Getenv("MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("MIGRATION_TEST_DSN não definido")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("abrir banco de teste: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("conectar ao banco de teste: %v", err)
	}
	if err := goose.SetDialect("mysql"); err != nil {
		t.Fatalf("configurar dialecto MySQL: %v", err)
	}
	goose.SetBaseFS(FS)

	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("aplicar migrations: %v", err)
	}

	assertTableExists(t, db, "formas_pagamento")
	assertTableExists(t, db, "pedido_pagamentos")
	assertColumnExists(t, db, "tenants", "waba_id")
	assertColumnExists(t, db, "tenants", "whatsapp_phone_id")
	assertIndexExists(t, db, "produtos", "idx_produtos_tenant_disponivel")
	assertIndexExists(t, db, "produtos", "idx_produtos_tenant_nome")
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
		table,
	).Scan(&count)
	if err != nil {
		t.Fatalf("verificar tabela %q: %v", table, err)
	}
	if count != 1 {
		t.Fatalf("tabela %q não existe", table)
	}
}

func assertColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?",
		table,
		column,
	).Scan(&count)
	if err != nil {
		t.Fatalf("verificar coluna %q.%q: %v", table, column, err)
	}
	if count != 1 {
		t.Fatalf("coluna %q.%q não existe", table, column)
	}
}

func assertIndexExists(t *testing.T, db *sql.DB, table, index string) {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?",
		table,
		index,
	).Scan(&count)
	if err != nil {
		t.Fatalf("verificar índice %q.%q: %v", table, index, err)
	}
	if count == 0 {
		t.Fatalf("índice %q.%q não existe", table, index)
	}
}
