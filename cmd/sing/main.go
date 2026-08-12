// scripts/test_signature.go
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func main() {
	// App Secret do .env
	secret := "4a5ff4eef0ee9ef06438c38ed34c6f04"

	// Corpo EXATO da requisição do Meta (copiado do log)
	body := `{"object":"whatsapp_business_account","entry":[{"id":"0","changes":[{"field":"messages","value":{"messaging_product":"whatsapp","metadata":{"display_phone_number":"16505551111","phone_number_id":"123456123"},"contacts":[{"profile":{"name":"test user name"},"wa_id":"16315551181","user_id":"US.13491208655302741918"}],"messages":[{"id":"ABGGFlA5Fpa","timestamp":"1504902988","from":"16315551181","from_user_id":"US.13491208655302741918","type":"text","text":{"body":"this is a text message"}}]}}]}]}`

	// Calcula a assinatura
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	assinaturaRecebida := "50249b2132e6f52e1a44d5882728d6374bd1cfecbbf708d9ce441c258ef5c982"

	fmt.Printf("Secret: %s\n", secret)
	fmt.Printf("Body: %s\n\n", body)
	fmt.Printf("Assinatura esperada: %s\n", expectedSignature)
	fmt.Printf("Assinatura recebida:  %s\n", assinaturaRecebida)
	fmt.Printf("Match: %t\n", expectedSignature == assinaturaRecebida)
}
