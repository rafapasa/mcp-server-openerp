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

func VerifySignature(r *http.Request, body []byte) error {
	signatureHeader := r.Header.Get("X-Hub-Signature-256")
	if signatureHeader == "" {
		return fmt.Errorf("assinatura X-Hub-Signature-256 não encontrada")
	}

	signature := strings.TrimPrefix(signatureHeader, "sha256=")
	if signature == "" {
		return fmt.Errorf("formato de assinatura inválido")
	}

	// Pega do ENV e limpa espaços/quebras de linha
	secret := strings.TrimSpace(os.Getenv("WHATSAPP_APP_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("WHATSAPP_APP_SECRET_FILE")) // se você usa _FILE no compose
	}
	if secret == "" {
		logger.GetLogger().Warn("WHATSAPP_APP_SECRET não configurado - pulando validação")
		return nil
	}

	// NUNCA loga o secret em produção
	// Calcula HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		// Loga só o erro, sem secret
		logger.GetLogger().Warn("Assinatura inválida do webhook",
			zap.String("expected", expectedSignature),
			zap.String("received", signature),
			zap.Int("body_len", len(body)),
		)
		return fmt.Errorf("assinatura inválida")
	}

	return nil
}

func VerifyWebhookRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error(r.Context(), "Erro ao ler body do webhook", zap.Error(err))
			http.Error(w, "Erro ao ler body", http.StatusBadRequest)
			return
		}
		// Restaura para o handler seguinte ler
		r.Body = io.NopCloser(bytes.NewReader(body))

		if err := VerifySignature(r, body); err != nil {
			logger.Warn(r.Context(), "Webhook não autorizado",
				zap.Error(err),
				zap.String("remote_addr", r.RemoteAddr),
			)
			http.Error(w, "Assinatura inválida", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func VerifyWebhookHandler(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return VerifyWebhookRequest(http.HandlerFunc(handler))
}
