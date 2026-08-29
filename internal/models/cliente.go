// internal/models/cliente.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// Cliente representa um cliente do sistema
type Cliente struct {
	ID         uint   `gorm:"primaryKey"`
	TenantID   uint   `gorm:"not null;uniqueIndex:idx_tenant_telefone"`
	Telefone   string `gorm:"type:varchar(20);not null;uniqueIndex:idx_tenant_telefone"`
	Nome       string `gorm:"type:varchar(100);comment:Nome completo do cliente"`
	NomePerfil string `gorm:"type:varchar(100);comment:Nome do perfil do WhatsApp"`
	Email      string `gorm:"type:varchar(100);comment:E-mail do cliente"`

	// Documentos
	InscricaoFederal   string `gorm:"type:varchar(20);comment:CPF (11 dígitos) ou CNPJ (14 dígitos)"`
	RG                 string `gorm:"type:varchar(20);comment:RG (apenas pessoa física)"`
	InscricaoEstadual  string `gorm:"type:varchar(20);comment:Inscrição Estadual (pessoa jurídica)"`
	InscricaoMunicipal string `gorm:"type:varchar(20);comment:Inscrição Municipal (pessoa jurídica)"`

	// Status
	Status          string     `gorm:"type:enum('ativo','inativo','pendente_validacao');default:ativo;comment:ativo, inativo, pendente_validacao"`
	StatusReason    string     `gorm:"type:varchar(255);comment:Motivo do status (ex: mudança de dono)"`
	StatusUpdatedAt *time.Time `gorm:"comment:Data da última mudança de status"`

	// Validação de Nome (mudança de dono)
	NomeAnterior        string     `gorm:"type:varchar(100);comment:Nome anterior (para validação)"`
	UltimaValidacaoNome *time.Time `gorm:"comment:Data da última validação de nome"`

	// Controle
	UltimoPedidoAt *time.Time `gorm:"comment:Data do último pedido"`
	CreatedAt      time.Time  `gorm:"autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime"`

	// Relacionamentos
	Tenant    Tenant     `gorm:"foreignKey:TenantID"`
	Enderecos []Endereco `gorm:"foreignKey:ClienteID"`
	Pedidos   []Pedido   `gorm:"foreignKey:ClienteID"`
}

// TableName retorna o nome da tabela
func (Cliente) TableName() string {
	return "clientes"
}

// ============================================
// MÉTODOS DE STATUS
// ============================================

// IsAtivo verifica se o cliente está ativo
func (c *Cliente) IsAtivo() bool {
	return c.Status == "ativo"
}

// IsInativo verifica se o cliente está inativo
func (c *Cliente) IsInativo() bool {
	return c.Status == "inativo"
}

// IsPendenteValidacao verifica se o cliente está pendente de validação
func (c *Cliente) IsPendenteValidacao() bool {
	return c.Status == "pendente_validacao"
}

// IsNomeValidado verifica se o nome do perfil já foi validado
func (c *Cliente) IsNomeValidado() bool {
	return c.UltimaValidacaoNome != nil
}

// ============================================
// MÉTODOS DE DOCUMENTO
// ============================================

// GetTipoPessoa determina o tipo de pessoa baseado no inscricao_federal
// Retorna: "fisica", "juridica" ou "desconhecido"
func (c *Cliente) GetTipoPessoa() string {
	if c.InscricaoFederal == "" {
		return "desconhecido"
	}

	// Remove caracteres não numéricos para contagem
	doc := c.InscricaoFederal
	if len(doc) == 11 {
		return "fisica"
	}
	if len(doc) == 14 {
		return "juridica"
	}
	return "desconhecido"
}

// TemDocumento verifica se o cliente possui documento fiscal
func (c *Cliente) TemDocumento() bool {
	return c.InscricaoFederal != ""
}

// ============================================
// MÉTODOS DE NOME
// ============================================

// NomeExibicao retorna o nome para exibição
// Prioriza o nome completo, fallback para nome_perfil
func (c *Cliente) NomeExibicao() string {
	if c.Nome != "" {
		return c.Nome
	}
	if c.NomePerfil != "" {
		return c.NomePerfil
	}
	return "Cliente"
}

// NomeCompleto retorna o nome completo ou nome_perfil se não disponível
func (c *Cliente) NomeCompleto() string {
	return c.NomeExibicao()
}

// ============================================
// MÉTODOS DE ENDEREÇO
// ============================================

// GetEnderecoPrincipal retorna o endereço principal do cliente
func (c *Cliente) GetEnderecoPrincipal() *Endereco {
	for _, e := range c.Enderecos {
		if e.Principal && e.IsAtivo() {
			return &e
		}
	}
	return nil
}

// GetEnderecosAtivos retorna todos os endereços ativos do cliente
func (c *Cliente) GetEnderecosAtivos() []Endereco {
	var ativos []Endereco
	for _, e := range c.Enderecos {
		if e.IsAtivo() {
			ativos = append(ativos, e)
		}
	}
	return ativos
}

// TemEnderecoPrincipal verifica se o cliente tem um endereço principal
func (c *Cliente) TemEnderecoPrincipal() bool {
	return c.GetEnderecoPrincipal() != nil
}

// ============================================
// MÉTODOS DE HISTÓRICO
// ============================================

// TemHistorico verifica se o cliente já fez pedidos
func (c *Cliente) TemHistorico() bool {
	return c.UltimoPedidoAt != nil
}

// DiasDesdeUltimoPedido retorna o número de dias desde o último pedido
func (c *Cliente) DiasDesdeUltimoPedido() int {
	if c.UltimoPedidoAt == nil {
		return -1
	}
	return int(time.Since(*c.UltimoPedidoAt).Hours() / 24)
}

// ============================================
// MÉTODOS DE ATUALIZAÇÃO
// ============================================

// AtualizarStatus atualiza o status do cliente
func (c *Cliente) AtualizarStatus(status, reason string) {
	c.Status = status
	c.StatusReason = reason
	now := time.Now()
	c.StatusUpdatedAt = &now
}

// AtualizarNome atualiza o nome do cliente e registra o anterior
func (c *Cliente) AtualizarNome(novoNome string) {
	if c.Nome != "" && c.Nome != novoNome {
		c.NomeAnterior = c.Nome
	}
	c.Nome = novoNome
	now := time.Now()
	c.UltimaValidacaoNome = &now
}

// AtualizarNomePerfil atualiza o nome do perfil
func (c *Cliente) AtualizarNomePerfil(novoPerfil string) {
	c.NomePerfil = novoPerfil
}

// AtualizarUltimoPedido atualiza a data do último pedido
func (c *Cliente) AtualizarUltimoPedido() {
	now := time.Now()
	c.UltimoPedidoAt = &now
}

// ============================================
// MÉTODOS DE VALIDAÇÃO DE NOME
// ============================================

// ValidarNome compara o nome perfil atual com o salvo
// Retorna true se o nome perfil for compatível com o nome salvo
// Usa similaridade > 80% (definido pelo chamador)
func (c *Cliente) ValidarNome(nomeAtual string, similaridade float64) bool {
	if similaridade >= 0.80 {
		return true
	}
	return false
}

// ============================================
// ENDERECO (MODEL ANINHADO)
// ============================================

// Endereco representa um endereço do cliente
type Endereco struct {
	ID        uint `gorm:"primaryKey"`
	ClienteID uint `gorm:"not null;index:idx_endereco_cliente"`

	// Endereço
	CEP         string `gorm:"type:varchar(10);index:idx_endereco_cep;comment:CEP (formato: 00000-000)"`
	Logradouro  string `gorm:"type:varchar(255);not null;comment:Nome da rua/avenida"`
	Numero      string `gorm:"type:varchar(20);not null;comment:Número do imóvel"`
	Complemento string `gorm:"type:varchar(100);comment:Complemento (apto, bloco, etc)"`
	Bairro      string `gorm:"type:varchar(100);comment:Bairro"`
	Cidade      string `gorm:"type:varchar(100);comment:Cidade"`
	Estado      string `gorm:"type:varchar(2);comment:UF (SP, RJ, etc)"`
	Pais        string `gorm:"type:varchar(50);default:Brasil;comment:País"`

	// Referência
	Referencia string `gorm:"type:varchar(255);comment:Ponto de referência"`

	// Geolocalização
	Latitude            float64 `gorm:"type:decimal(10,8);comment:Latitude (ex: -23.550520)"`
	Longitude           float64 `gorm:"type:decimal(11,8);comment:Longitude (ex: -46.633308)"`
	GeolocalizacaoFonte string  `gorm:"type:varchar(50);comment:Fonte: whatsapp, viacep, google, manual"`

	// Tipo e Padrão
	Tipo      string `gorm:"type:enum('residencial','comercial','entrega','cobranca');default:entrega"`
	Principal bool   `gorm:"default:false;comment:Endereço principal para entregas"`

	// Soft Delete
	DeletedAt gorm.DeletedAt `gorm:"index:idx_endereco_deleted_at;comment:Data de exclusão lógica"`

	// Metadados
	Observacoes string    `gorm:"type:text;comment:Observações sobre o endereço"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`

	// Relacionamentos
	Cliente Cliente  `gorm:"foreignKey:ClienteID"`
	Pedidos []Pedido `gorm:"foreignKey:EnderecoEntregaID"`
}

// TableName retorna o nome da tabela
func (Endereco) TableName() string {
	return "enderecos"
}

// IsPrincipal verifica se é o endereço principal
func (e *Endereco) IsPrincipal() bool {
	return e.Principal
}

// IsAtivo verifica se o endereço está ativo (não deletado)
func (e *Endereco) IsAtivo() bool {
	return e.DeletedAt.Time.IsZero()
}

// EnderecoCompleto retorna o endereço formatado como string
func (e *Endereco) EnderecoCompleto() string {
	if e.Logradouro == "" {
		return ""
	}

	result := e.Logradouro + ", " + e.Numero
	if e.Complemento != "" {
		result += " - " + e.Complemento
	}
	if e.Bairro != "" {
		result += " - " + e.Bairro
	}
	result += ", " + e.Cidade + " - " + e.Estado
	if e.CEP != "" {
		result += ", CEP: " + e.CEP
	}
	return result
}

// EnderecoResumido retorna endereço resumido para exibição
func (e *Endereco) EnderecoResumido() string {
	if e.Logradouro == "" {
		return ""
	}
	result := e.Logradouro + ", " + e.Numero
	if e.Complemento != "" {
		result += " " + e.Complemento
	}
	if e.Bairro != "" {
		result += " - " + e.Bairro
	}
	return result
}

// HasGeolocalizacao verifica se o endereço tem coordenadas
func (e *Endereco) HasGeolocalizacao() bool {
	return e.Latitude != 0 || e.Longitude != 0
}
