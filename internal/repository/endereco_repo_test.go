// internal/repository/endereco_repo_test.go
package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	dsnRoot := "root:root@tcp(127.0.0.1:3306)/?charset=utf8mb4&parseTime=True&loc=Local"
	dbRoot, err := gorm.Open(mysql.Open(dsnRoot), &gorm.Config{})
	if err != nil {
		t.Fatalf("Erro ao conectar no MySQL: %v", err)
	}

	if err := dbRoot.Exec("CREATE DATABASE IF NOT EXISTS test_db").Error; err != nil {
		t.Fatalf("Erro ao criar banco de teste: %v", err)
	}
	sqlDBRoot, _ := dbRoot.DB()
	sqlDBRoot.Close()

	dsn := "root:root@tcp(127.0.0.1:3306)/test_db?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Erro ao conectar no banco de teste: %v", err)
	}

	if err := db.AutoMigrate(
		&models.Cliente{},
		&models.Endereco{},
		&models.Tenant{},
	); err != nil {
		t.Fatalf("Erro ao migrar tabelas: %v", err)
	}

	tenant := models.Tenant{
		ID:        1,
		Nome:      "Teste",
		CNPJ:      "12.345.678/0001-90",
		Telefone:  "(47) 99999-9999",
		Endereco:  "Rua Teste, 123 - Centro, Pinhalzinho-SC",
		Segmento:  "restaurante",
		Ativo:     true,
		CreatedAt: time.Now(),
	}
	if err := db.Where("id = ?", tenant.ID).FirstOrCreate(&tenant).Error; err != nil {
		t.Fatalf("Erro ao criar tenant de teste: %v", err)
	}

	// Teardown: roda automaticamente ao fim do teste, mesmo se ele falhar
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err != nil {
			return
		}
		// Opção A: só fecha a conexão
		// sqlDB.Close()

		// Opção B (mais comum em testes de integração): limpa as tabelas
		db.Exec("DELETE FROM enderecos")
		db.Exec("DELETE FROM clientes")
		db.Exec("DELETE FROM tenants")
		sqlDB.Close()
	})

	return db
}

func TestEnderecoRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEnderecoRepository(db)
	clienteRepo := NewClienteRepository(db)

	ctx := context.Background()

	// Cria um cliente
	cliente := &models.Cliente{
		TenantID: 1,
		Telefone: "5511999999999",
		Nome:     "João Silva",
		Status:   "ativo",
	}
	err := clienteRepo.Create(ctx, cliente)
	if err != nil {
		fmt.Printf("Erro: %v", err)
	}
	assert.NoError(t, err)

	// Cria um endereço
	endereco := &models.Endereco{
		ClienteID:  cliente.ID,
		Logradouro: "Rua das Flores",
		Numero:     "123",
		Bairro:     "Jardim Paulista",
		Cidade:     "São Paulo",
		Estado:     "SP",
		CEP:        "01234-567",
		Principal:  true,
	}

	err = repo.Create(ctx, endereco)
	assert.NoError(t, err)
	assert.NotZero(t, endereco.ID)

	// Verifica se foi salvo
	enderecoSalvo, err := repo.FindByID(ctx, endereco.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Rua das Flores", enderecoSalvo.Logradouro)
	assert.True(t, enderecoSalvo.Principal)
}
