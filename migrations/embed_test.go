package migrations

import (
	"strings"
	"testing"
)

func TestFSEmbedContemMigracoesGoose(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("ler migrations.FS: %v", err)
	}

	var migrationCount int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		migrationCount++

		content, err := FS.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("ler migração %q: %v", entry.Name(), err)
		}
		sql := string(content)
		if !strings.Contains(sql, "-- +goose Up") {
			t.Errorf("migração %q não contém a diretiva -- +goose Up", entry.Name())
		}
		if strings.Contains(sql, "CREATE") && !strings.Contains(sql, "-- +goose StatementBegin") {
			t.Errorf("migração %q contém SQL sem -- +goose StatementBegin", entry.Name())
		}
	}
	if migrationCount != 6 {
		t.Fatalf("quantidade inesperada de migrations: got %d, want 6", migrationCount)
	}
}
