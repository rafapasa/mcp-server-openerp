package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ============================================
// JSON Custom Types para GORM
// ============================================

// JSONArray para campos como ingredientes
type JSONArray []string

func (j JSONArray) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// JSONMap para campos genéricos
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// ============================================
// MODELS
// ============================================

// Tenant representa um restaurante
type Tenant struct {
	ID        uint      `gorm:"primaryKey"`
	Nome      string    `gorm:"size:100;not null"`
	CNPJ      string    `gorm:"size:18"`
	Telefone  string    `gorm:"size:20"`
	Endereco  string    `gorm:"size:255"`
	Ativo     bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Relacionamentos
	Categorias []Categoria `gorm:"foreignKey:TenantID"`
	Produtos   []Produto   `gorm:"foreignKey:TenantID"`
	Pedidos    []Pedido    `gorm:"foreignKey:TenantID"`
}

func (Tenant) TableName() string {
	return "tenants"
}

// Categoria representa uma categoria de produtos
type Categoria struct {
	ID        uint      `gorm:"primaryKey"`
	TenantID  uint      `gorm:"not null;index"`
	Nome      string    `gorm:"size:50;not null"`
	Ordem     int       `gorm:"default:0"`
	Ativo     bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Relacionamentos
	Tenant   Tenant    `gorm:"foreignKey:TenantID"`
	Produtos []Produto `gorm:"foreignKey:CategoriaID"`
}

func (Categoria) TableName() string {
	return "categorias"
}

// Produto representa um item do cardápio
type Produto struct {
	ID           uint      `gorm:"primaryKey"`
	TenantID     uint      `gorm:"not null;index"`
	CategoriaID  *uint     `gorm:"index"` // Pode ser nulo
	Nome         string    `gorm:"size:100;not null"`
	Descricao    string    `gorm:"size:255"`
	Preco        float64   `gorm:"type:decimal(10,2);not null"`
	Ingredientes JSONArray `gorm:"type:json"`
	Disponivel   bool      `gorm:"default:true"`
	TempoPreparo int       `gorm:"default:15"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`

	// Relacionamentos
	Tenant    Tenant     `gorm:"foreignKey:TenantID"`
	Categoria *Categoria `gorm:"foreignKey:CategoriaID"`
}

func (Produto) TableName() string {
	return "produtos"
}

// ItemPedido representa um item dentro do pedido (JSON)
type ItemPedido struct {
	ProdutoID  uint    `json:"produto_id"`
	Nome       string  `json:"nome"`
	Quantidade int     `json:"qtd"`
	Preco      float64 `json:"preco"`
	Observacao string  `json:"obs,omitempty"`
}

// Pedido representa um pedido feito pelo WhatsApp
type Pedido struct {
	ID              uint            `gorm:"primaryKey"`
	TenantID        uint            `gorm:"not null;index"`
	ClienteID       string          `gorm:"size:50;index"`
	ClienteNome     string          `gorm:"size:100"`
	ClienteTelefone string          `gorm:"size:20"`
	Itens           json.RawMessage `gorm:"type:json;not null"` // Array de ItemPedido
	Total           float64         `gorm:"type:decimal(10,2);not null"`
	Status          string          `gorm:"type:enum('pendente','confirmado','preparando','entregue','cancelado');default:'pendente'"`
	Observacoes     string          `gorm:"type:text"`
	TempoEstimado   int             `gorm:"default:0"`
	Origem          string          `gorm:"type:enum('whatsapp','dashboard','api');default:'whatsapp'"`
	CreatedAt       time.Time       `gorm:"autoCreateTime;index:idx_pedidos_tenant_created,priority:2"`
	UpdatedAt       time.Time       `gorm:"autoUpdateTime"`

	// Relacionamentos
	Tenant Tenant `gorm:"foreignKey:TenantID"`
}

func (Pedido) TableName() string {
	return "pedidos"
}

// ============================================
// CONSTANTES
// ============================================

const (
	StatusPendente   = "pendente"
	StatusConfirmado = "confirmado"
	StatusPreparando = "preparando"
	StatusEntregue   = "entregue"
	StatusCancelado  = "cancelado"
)

const (
	OrigemWhatsApp  = "whatsapp"
	OrigemDashboard = "dashboard"
	OrigemAPI       = "api"
)
