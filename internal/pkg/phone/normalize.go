// internal/pkg/phone/normalize.go
package phone

import "regexp"

var nonDigits = regexp.MustCompile(`\D`)

// Normalize remove tudo que não é dígito
// +55 (49) 8901-4080 -> 554989014080
// 1337919362728144 continua 1337919362728144 (usado pra log)
func Normalize(raw string) string {
	if raw == "" {
		return ""
	}
	return nonDigits.ReplaceAllString(raw, "")
}

// NormalizeBR garante DDI 55 se vier só DDD+numero
func NormalizeBR(raw string) string {
	n := Normalize(raw)
	if len(n) == 11 || len(n) == 10 {
		return "55" + n
	}
	return n
}
