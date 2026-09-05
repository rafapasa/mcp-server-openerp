package migrations

import "testing"

func TestFSEmbedContemMigracoes(t *testing.T) {
	want := []string{
		"000001_create_tenants.up.sql",
		"000002_create_categorias.up.sql",
		"000003_create_produtos.up.sql",
		"000004_add_whatsapp_fields_to_tenants.up.sql",
		"000005_add_produtos_indexes.up.sql",
	}
	for _, name := range want {
		if _, err := FS.ReadFile(name); err != nil {
			t.Errorf("migração %q não embutida em migrations.FS: %v", name, err)
		}
	}
}
