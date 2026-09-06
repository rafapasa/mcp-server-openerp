package repository

import (
	"context"
	"testing"

	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:user-repository-test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL,
			nome TEXT NOT NULL,
			email TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'admin',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestUserRepositoryFindByEmail(t *testing.T) {
	db := setupUserRepositoryTestDB(t)
	require.NoError(t, db.Create(&models.User{
		TenantID: 1, Nome: "Ativo", Email: "etoolstec@etoolstec.com.br",
		PasswordHash: "hash", Role: "admin", IsActive: true,
	}).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO users (tenant_id, nome, email, password_hash, role, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`, 1, "Inativo", "inativo@etoolstec.com.br", "hash", "admin", false).Error)
	require.NoError(t, db.Create(&models.User{
		TenantID: 2, Nome: "Outro tenant", Email: "etoolstec@etoolstec.com.br",
		PasswordHash: "hash", Role: "admin", IsActive: true,
	}).Error)

	repo := NewUserRepository(db)

	t.Run("retorna somente usuário ativo do tenant", func(t *testing.T) {
		users, err := repo.FindByEmail(context.Background(), 1, "etoolstec@etoolstec.com.br")
		require.NoError(t, err)
		require.Len(t, *users, 1)
		require.Equal(t, "Ativo", (*users)[0].Nome)
	})

	t.Run("não retorna usuário inativo", func(t *testing.T) {
		users, err := repo.FindByEmail(context.Background(), 1, "inativo@etoolstec.com.br")
		require.NoError(t, err)
		require.Empty(t, *users)
	})
}
