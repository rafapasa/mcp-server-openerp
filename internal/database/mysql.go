package database

import (
	"fmt"
	"log"
	"time"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQL representa a conexão com o banco de dados
type MySQL struct {
	DB *gorm.DB
}

// NewMySQL cria uma nova conexão com o MySQL
func NewMySQL(cfg *config.Config, dsn ...string) (*MySQL, error) {
	isTest := len(dsn) > 0 && dsn[0] != ""

	var logLevel logger.LogLevel
	switch cfg.Environment {
	case "development":
		if isTest {
			logLevel = logger.Warn
		} else {
			logLevel = logger.Info
		}
	case "production":
		logLevel = logger.Error
	default:
		logLevel = logger.Silent
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
		SkipDefaultTransaction:                   true,
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: isTest,
	}

	// Abrir conexão
	db, err := gorm.Open(mysql.Open(getConnectionString(cfg, dsn...)), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar ao banco de dados: %w", err)
	}

	// Obter a instância do *sql.DB para configurar o pool de conexões
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter o *sql.DB: %w", err)
	}

	// Configurar o pool de conexões
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Testar a conexão
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao testar conexão com o banco: %w", err)
	}

	log.Println("✅ Conectado ao banco de dados MySQL com sucesso!")
	log.Printf("📊 Database: %s @ %s:%s", cfg.DBName, cfg.DBHost, cfg.DBPort)

	return &MySQL{DB: db}, nil
}

func getConnectionString(cfg *config.Config, dsn ...string) string {
	if len(dsn) > 0 && dsn[0] != "" {
		return dsn[0]
	} else {
		return cfg.GetDSN()
	}
}

// Close fecha a conexão com o banco de dados
func (m *MySQL) Close() error {
	sqlDB, err := m.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping verifica se a conexão está ativa
func (m *MySQL) Ping() error {
	sqlDB, err := m.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// IsConnected verifica se está conectado ao banco
func (m *MySQL) IsConnected() bool {
	return m.Ping() == nil
}

// WithTransaction executa uma função dentro de uma transação
func (m *MySQL) WithTransaction(fn func(tx *gorm.DB) error) error {
	return m.DB.Transaction(fn)
}

// GetDB retorna a instância do GORM
func (m *MySQL) GetDB() *gorm.DB {
	return m.DB
}
