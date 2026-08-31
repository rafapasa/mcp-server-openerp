package dto

type TenantDTO struct {
	ID                    uint   `json:"id"`
	Nome                  string `json:"nome"`
	CNPJ                  string `json:"cnpj"`
	Telefone              string `json:"telefone"`
	Endereco              string `json:"endereco"`
	Segmento              string `json:"segmento"` // farmacia, mercado, restaurante, geral
	WabaID                string `json:"waba_id"`
	WhatsappPhoneID       string `json:"whatsapp_phone_id"`
	WhatsappDisplayNumber string `json:"whatsapp_display_number"`
	Ativo                 bool   `json:"ativo"`
}

type CreateTenantDTO struct {
	Nome                  string `json:"nome" validate:"required"`
	CNPJ                  string `json:"cnpj"`
	Telefone              string `json:"telefone"`
	Segmento              string `json:"segmento"`
	WabaID                string `json:"waba_id"`
	WhatsappPhoneID       string `json:"whatsapp_phone_id" validate:"required"`
	WhatsappDisplayNumber string `json:"whatsapp_display_number"`
}
