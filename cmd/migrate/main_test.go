package main

import (
	"database/sql"
	"strings"
	"testing"
)

func TestPedidoStatusMigrationStatementsFromLegacyEnum(t *testing.T) {
	stmts := pedidoStatusMigrationStatements(
		"enum('pendente','confirmado','preparando','entregue','cancelado')",
		"YES",
		sql.NullString{String: "pendente", Valid: true},
	)
	if len(stmts) != 3 {
		t.Fatalf("esperava 3 comandos, got %d: %v", len(stmts), stmts)
	}

	if !strings.HasPrefix(stmts[0], "ALTER TABLE `pedidos` MODIFY COLUMN `status` ENUM(") {
		t.Errorf("primeiro comando deveria ampliar o ENUM, got %q", stmts[0])
	}
	if !strings.Contains(stmts[0], "'preparando'") {
		t.Errorf("ENUM ampliado deve manter 'preparando' para não truncar as linhas existentes, got %q", stmts[0])
	}
	if !strings.Contains(stmts[0], "'em_preparo'") || !strings.Contains(stmts[0], "'saiu_para_entrega'") {
		t.Errorf("ENUM ampliado deveria conter os novos status, got %q", stmts[0])
	}

	if !strings.Contains(stmts[1], "UPDATE") ||
		!strings.Contains(stmts[1], "'preparando'") ||
		!strings.Contains(stmts[1], "'em_preparo'") {
		t.Errorf("segundo comando deveria remapear preparando -> em_preparo, got %q", stmts[1])
	}

	if !strings.HasPrefix(stmts[2], "ALTER TABLE `pedidos` MODIFY COLUMN `status` ENUM(") {
		t.Errorf("terceiro comando deveria estreitar o ENUM, got %q", stmts[2])
	}
	if strings.Contains(stmts[2], "'preparando'") {
		t.Errorf("ENUM final não deveria conter 'preparando', got %q", stmts[2])
	}
}

func TestPedidoStatusMigrationStatementsJaMigrado(t *testing.T) {
	stmts := pedidoStatusMigrationStatements(
		"enum('pendente','confirmado','em_preparo','saiu_para_entrega','entregue','cancelado')",
		"YES",
		sql.NullString{String: "pendente", Valid: true},
	)
	if len(stmts) != 0 {
		t.Fatalf("coluna já migrada deveria gerar 0 comandos, got %d: %v", len(stmts), stmts)
	}
}

func TestPedidoStatusMigrationStatementsReconheceEnumMaiusculo(t *testing.T) {
	stmts := pedidoStatusMigrationStatements(
		"ENUM('pendente','confirmado','em_preparo','saiu_para_entrega','entregue','cancelado')",
		"YES",
		sql.NullString{},
	)
	if len(stmts) != 0 {
		t.Fatalf("deveria reconhecer ENUM em maiúsculas como já migrado, got %v", stmts)
	}
}

func TestPedidoStatusMigrationStatementsComPreparandoRemanescente(t *testing.T) {
	stmts := pedidoStatusMigrationStatements(
		"enum('pendente','confirmado','preparando','em_preparo','saiu_para_entrega','entregue','cancelado')",
		"YES",
		sql.NullString{},
	)
	if len(stmts) != 3 {
		t.Fatalf("esperava 3 comandos para remover 'preparando' remanescente, got %d: %v", len(stmts), stmts)
	}
}

func TestPedidoStatusMigrationStatementsPreservaDefinicaoDaColuna(t *testing.T) {
	notNull := pedidoStatusMigrationStatements(
		"enum('pendente','confirmado','preparando','entregue','cancelado')",
		"NO",
		sql.NullString{String: "pendente", Valid: true},
	)
	for _, stmt := range []string{notNull[0], notNull[2]} {
		if !strings.Contains(stmt, "NOT NULL") {
			t.Errorf("ALTER deveria preservar NOT NULL, got %q", stmt)
		}
		if !strings.Contains(stmt, "DEFAULT 'pendente'") {
			t.Errorf("ALTER deveria preservar o DEFAULT, got %q", stmt)
		}
	}

	nullable := pedidoStatusMigrationStatements(
		"enum('pendente','confirmado','preparando','entregue','cancelado')",
		"YES",
		sql.NullString{},
	)
	for _, stmt := range []string{nullable[0], nullable[2]} {
		if strings.Contains(stmt, "NOT NULL") {
			t.Errorf("ALTER não deveria introduzir NOT NULL, got %q", stmt)
		}
		if strings.Contains(stmt, "DEFAULT") {
			t.Errorf("ALTER não deveria introduzir DEFAULT, got %q", stmt)
		}
	}
}
