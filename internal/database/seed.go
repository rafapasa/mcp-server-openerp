// internal/database/seed.go
package database

import (
	"log"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"gorm.io/gorm"
)

// Seed popula o banco com dados iniciais
func Seed(db *gorm.DB) error {
	log.Println("🌱 Inserindo dados iniciais...")

	// 1. Cria Tenant
	tenant := models.Tenant{
		Nome:     "FastFood do Zé",
		CNPJ:     "12.345.678/0001-90",
		Telefone: "(11) 99999-9999",
		Endereco: "Rua das Flores, 123",
		Ativo:    true,
	}

	if err := db.FirstOrCreate(&tenant, models.Tenant{Nome: "FastFood do Zé"}).Error; err != nil {
		return err
	}

	// 2. Cria Categorias
	categorias := []models.Categoria{
		{TenantID: tenant.ID, Nome: "Lanches", Ordem: 1, Ativo: true},
		{TenantID: tenant.ID, Nome: "Bebidas", Ordem: 2, Ativo: true},
		{TenantID: tenant.ID, Nome: "Acompanhamentos", Ordem: 3, Ativo: true},
		{TenantID: tenant.ID, Nome: "Sobremesas", Ordem: 4, Ativo: true},
	}

	for _, cat := range categorias {
		if err := db.FirstOrCreate(&cat, models.Categoria{TenantID: tenant.ID, Nome: cat.Nome}).Error; err != nil {
			return err
		}
	}

	// Busca as categorias criadas
	var cats []models.Categoria
	db.Where("tenant_id = ?", tenant.ID).Find(&cats)

	// Mapeia por nome
	catMap := make(map[string]uint)
	for _, c := range cats {
		catMap[c.Nome] = c.ID
	}

	// 3. Cria Produtos
	produtos := []models.Produto{
		{
			TenantID:     tenant.ID,
			CategoriaID:  ptr(catMap["Lanches"]),
			Nome:         "X-Bacon",
			Descricao:    "Pão, hambúrguer, bacon, queijo, alface, tomate",
			Preco:        29.90,
			Ingredientes: models.JSONArray{"pão", "hambúrguer", "bacon", "queijo"},
			Disponivel:   true,
			TempoPreparo: 15,
		},
		{
			TenantID:     tenant.ID,
			CategoriaID:  ptr(catMap["Lanches"]),
			Nome:         "X-Tudo",
			Descricao:    "Pão, hambúrguer, bacon, ovo, queijo, alface, tomate",
			Preco:        34.90,
			Ingredientes: models.JSONArray{"pão", "hambúrguer", "bacon", "ovo", "queijo"},
			Disponivel:   true,
			TempoPreparo: 18,
		},
		{
			TenantID:     tenant.ID,
			CategoriaID:  ptr(catMap["Lanches"]),
			Nome:         "Hambúrguer Simples",
			Descricao:    "Pão, hambúrguer, queijo",
			Preco:        19.90,
			Ingredientes: models.JSONArray{"pão", "hambúrguer", "queijo"},
			Disponivel:   true,
			TempoPreparo: 10,
		},
		{
			TenantID:     tenant.ID,
			CategoriaID:  ptr(catMap["Bebidas"]),
			Nome:         "Coca-Cola 2L",
			Descricao:    "Refrigerante Coca-Cola 2 litros",
			Preco:        12.00,
			Ingredientes: models.JSONArray{"água", "açúcar", "cafeína"},
			Disponivel:   true,
			TempoPreparo: 0,
		},
		{
			TenantID:     tenant.ID,
			CategoriaID:  ptr(catMap["Bebidas"]),
			Nome:         "Suco de Laranja",
			Descricao:    "Suco natural de laranja 500ml",
			Preco:        8.00,
			Ingredientes: models.JSONArray{"laranja", "água"},
			Disponivel:   true,
			TempoPreparo: 0,
		},
		{
			TenantID:     tenant.ID,
			CategoriaID:  ptr(catMap["Acompanhamentos"]),
			Nome:         "Batata Frita",
			Descricao:    "Batata frita crocante porção média",
			Preco:        14.00,
			Ingredientes: models.JSONArray{"batata", "sal", "óleo"},
			Disponivel:   true,
			TempoPreparo: 10,
		},
		{
			TenantID:     tenant.ID,
			CategoriaID:  ptr(catMap["Acompanhamentos"]),
			Nome:         "Onion Rings",
			Descricao:    "Anéis de cebola empanados",
			Preco:        12.00,
			Ingredientes: models.JSONArray{"cebola", "farinha", "óleo"},
			Disponivel:   true,
			TempoPreparo: 10,
		},
		{
			TenantID:     tenant.ID,
			CategoriaID:  ptr(catMap["Sobremesas"]),
			Nome:         "Milkshake",
			Descricao:    "Milkshake de chocolate 500ml",
			Preco:        15.00,
			Ingredientes: models.JSONArray{"leite", "sorvete", "chocolate"},
			Disponivel:   true,
			TempoPreparo: 5,
		},
	}

	for _, p := range produtos {
		if err := db.FirstOrCreate(&p, models.Produto{TenantID: tenant.ID, Nome: p.Nome}).Error; err != nil {
			return err
		}
	}

	log.Printf("✅ Seed concluído! Tenant ID: %d", tenant.ID)
	return nil
}

// Helper para ponteiro de uint
func ptr(v uint) *uint {
	return &v
}
