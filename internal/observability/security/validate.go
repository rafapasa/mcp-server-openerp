// internal/observability/security/validate.go
package security

import (
	"fmt"
	"regexp"
	"strconv"
)

// ValidateTenantID valida o ID do tenant
func ValidateTenantID(tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id não pode ser vazio")
	}

	// Apenas números, letras, underscore e hífen
	valid := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !valid.MatchString(tenantID) {
		return fmt.Errorf("tenant_id inválido: %s", tenantID)
	}

	return nil
}

// ValidateClientID valida o ID do cliente (número de telefone)
func ValidatePhoneNumberID(phoneNumberID string) error {
	if phoneNumberID == "" {
		return fmt.Errorf("client_id não pode ser vazio")
	}

	// Número de telefone: 10-15 dígitos
	valid := regexp.MustCompile(`^[0-9]{10,15}$`)
	if !valid.MatchString(phoneNumberID) {
		return fmt.Errorf("client_id inválido (deve ter 10-15 dígitos): %s", phoneNumberID  )
	}

	return nil
}

// ValidateMessage valida uma mensagem
func ValidateMessage(msg string) error {
	if msg == "" {
		return fmt.Errorf("mensagem não pode ser vazia")
	}

	if len(msg) > 5000 {
		return fmt.Errorf("mensagem muito longa (máx: 5000 caracteres)")
	}

	return nil
}

// SanitizeAndValidate sanitiza e valida uma mensagem
func SanitizeAndValidate(msg string) (string, error) {
	if err := ValidateMessage(msg); err != nil {
		return "", err
	}

	sanitized := SanitizeMessage(msg)
	return sanitized, nil
}

// ParseTenantID converte tenant_id string para uint
func ParseTenantID(tenantID string) (uint, error) {
	if err := ValidateTenantID(tenantID); err != nil {
		return 0, err
	}

	id, err := strconv.ParseUint(tenantID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("tenant_id deve ser um número: %w", err)
	}

	return uint(id), nil
}
