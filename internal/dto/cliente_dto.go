// internal/dto/cliente_dto.go
package dto

import "time"

// ClienteDTO representa os dados de um cliente para APIs
type ClienteDTO struct {
	ID                  uint          `json:"id"`
	TenantID            uint          `json:"tenant_id"`
	Telefone            string        `json:"telefone"`
	Nome                string        `json:"nome"`
	NomePerfil          string        `json:"nome_perfil"`
	Email               string        `json:"email"`
	InscricaoFederal    string        `json:"inscricao_federal,omitempty"`
	RG                  string        `json:"rg,omitempty"`
	InscricaoEstadual   string        `json:"inscricao_estadual,omitempty"`
	InscricaoMunicipal  string        `json:"inscricao_municipal,omitempty"`
	Status              string        `json:"status"`
	StatusReason        string        `json:"status_reason,omitempty"`
	StatusUpdatedAt     *time.Time    `json:"status_updated_at,omitempty"`
	NomeAnterior        string        `json:"nome_anterior,omitempty"`
	UltimaValidacaoNome *time.Time    `json:"ultima_validacao_nome,omitempty"`
	UltimoPedidoAt      *time.Time    `json:"ultimo_pedido_at,omitempty"`
	Enderecos           []EnderecoDTO `json:"enderecos,omitempty"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

// EnderecoDTO representa os dados de um endereço para APIs
type EnderecoDTO struct {
	ID          uint      `json:"id"`
	ClienteID   uint      `json:"cliente_id"`
	CEP         string    `json:"cep,omitempty"`
	Logradouro  string    `json:"logradouro"`
	Numero      string    `json:"numero"`
	Complemento string    `json:"complemento,omitempty"`
	Bairro      string    `json:"bairro,omitempty"`
	Cidade      string    `json:"cidade,omitempty"`
	Estado      string    `json:"estado,omitempty"`
	Pais        string    `json:"pais,omitempty"`
	Referencia  string    `json:"referencia,omitempty"`
	Latitude    float64   `json:"latitude,omitempty"`
	Longitude   float64   `json:"longitude,omitempty"`
	Tipo        string    `json:"tipo"`
	Principal   bool      `json:"principal"`
	Observacoes string    `json:"observacoes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CriarClienteRequest representa a requisição para criar um cliente
type CriarClienteRequest struct {
	TenantID         uint   `json:"tenant_id"`
	Telefone         string `json:"telefone"`
	Nome             string `json:"nome"`
	NomePerfil       string `json:"nome_perfil"`
	Email            string `json:"email"`
	InscricaoFederal string `json:"inscricao_federal,omitempty"`
}

// AtualizarClienteRequest representa a requisição para atualizar um cliente
type AtualizarClienteRequest struct {
	Nome             string `json:"nome,omitempty"`
	NomePerfil       string `json:"nome_perfil,omitempty"`
	Email            string `json:"email,omitempty"`
	InscricaoFederal string `json:"inscricao_federal,omitempty"`
	Observacoes      string `json:"observacoes,omitempty"`
}

// CriarEnderecoRequest representa a requisição para criar um endereço
type CriarEnderecoRequest struct {
	CEP         string  `json:"cep,omitempty"`
	Logradouro  string  `json:"logradouro"`
	Numero      string  `json:"numero"`
	Complemento string  `json:"complemento,omitempty"`
	Bairro      string  `json:"bairro,omitempty"`
	Cidade      string  `json:"cidade,omitempty"`
	Estado      string  `json:"estado,omitempty"`
	Pais        string  `json:"pais,omitempty"`
	Referencia  string  `json:"referencia,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Tipo        string  `json:"tipo,omitempty"`
	Principal   bool    `json:"principal"`
}
