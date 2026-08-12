// internal/webhook/security.go
package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

// VerifySignature verifica a assinatura do webhook do Meta
// Retorna erro se a assinatura for inválida ou não encontrada
func VerifySignature(r *http.Request, body []byte) error {
	// 1. Pega a assinatura do header
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return fmt.Errorf("assinatura X-Hub-Signature-256 não encontrada no header")
	}

	// 2. Remove o prefixo "sha256="
	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return fmt.Errorf("formato de assinatura inválido (esperado: sha256=<hash>)")
	}

	// 3. Pega o secret do ambiente
	secret := os.Getenv("WHATSAPP_APP_SECRET")
	if secret == "" {
		// Em desenvolvimento, permite continuar sem secret (com warn)
		logger.GetLogger().Warn("WHATSAPP_APP_SECRET não configurado - assinatura não será verificada",
			zap.String("warning", "Configure WHATSAPP_APP_SECRET no .env para produção"),
		)
		return nil
	}

	logger.GetLogger().Info("Validação de Assinatura de requisição",
		zap.String("secret", secret),
		zap.String("signature", signature),
		zap.ByteString("body", body),
	)

	// 4. Calcula a assinatura esperada
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// 5. Compara de forma segura (constant-time)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("assinatura inválida: esperado %s, recebido %s", expectedSignature, signature)
	}

	// 6. Sucesso! ✅
	return nil
}

// VerifyWebhookRequest é um middleware que verifica a assinatura do webhook
// Deve ser usado apenas nas rotas POST do webhook
func VerifyWebhookRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Só verifica requisições POST (webhooks)
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		// Lê o body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error(r.Context(), "Erro ao ler body do webhook",
				zap.Error(err),
			)
			http.Error(w, "Erro ao ler body", http.StatusBadRequest)
			return
		}

		logger.Info(r.Context(), "Verificando assinatura do webhook",
			zap.ByteString("body", body),
		)

		// Restaura o body para uso posterior
		r.Body = io.NopCloser(bytes.NewReader(body))

		// Verifica a assinatura
		if err := VerifySignature(r, body); err != nil {
			logger.Warn(r.Context(), "Assinatura do webhook inválida",
				zap.Error(err),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
			)
			http.Error(w, "Assinatura inválida", http.StatusUnauthorized)
			return
		}

		logger.Debug(r.Context(), "Assinatura do webhook verificada com sucesso")

		// Prossegue para o handler
		next.ServeHTTP(w, r)
	}
}

// VerifyWebhookHandler é um wrapper que pode ser usado diretamente no handler
func VerifyWebhookHandler(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return VerifyWebhookRequest(http.HandlerFunc(handler))
}
