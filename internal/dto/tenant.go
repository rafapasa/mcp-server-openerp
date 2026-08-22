package dto

type TenantDTO struct {
	ID       uint   `json:"id"`
	Nome     string `json:"nome"`
	CNPJ     string `json:"cnpj"`
	Telefone string `json:"telefone"`
	Endereco string `json:"endereco"`
	Segmento string `json:"segmento"` // farmacia, autopecas, restaurante, contabilidade
	Ativo    bool   `json:"ativo"`
}

type CreateTenantDTO struct {
	Nome     string `json:"nome" validate:"required"`
	CNPJ     string `json:"cnpj"`
	Telefone string `json:"telefone"`
	Segmento string `json:"segmento"`
}
