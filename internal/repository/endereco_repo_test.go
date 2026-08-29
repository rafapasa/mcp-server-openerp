// internal/repository/endereco_repo_test.go
package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// MySQL rodando localmente (via Docker)
	dsn := "root:root@tcp(127.0.0.1:3306)/test_db?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Erro ao criar banco de teste: %v", err)
	}

	// Cria banco se não existir
	db.Exec("CREATE DATABASE IF NOT EXISTS test_db")
	db.Exec("USE test_db")

	// Migra TUDO (suporta ENUM)
	db.AutoMigrate(&models.Cliente{}, &models.Endereco{})

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
