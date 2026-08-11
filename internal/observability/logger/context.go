// internal/observability/logger/context.go
package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type ctxKey string

const (
	// LoggerKey é a chave para o logger no contexto
	LoggerKey ctxKey = "logger"
	// TraceIDKey é a chave para o trace_id no contexto
	TraceIDKey ctxKey = "trace_id"
	// SpanIDKey é a chave para o span_id no contexto
	SpanIDKey ctxKey = "span_id"
)

// WithLogger adiciona um logger ao contexto
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// FromContext retorna o logger do contexto ou o logger global
func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*zap.Logger); ok {
		return logger
	}
	return GetLogger()
}

// WithFields adiciona campos ao logger do contexto
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	logger := FromContext(ctx)
	return WithLogger(ctx, logger.With(fields...))
}

// WithTraceID adiciona trace_id ao logger do contexto
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return WithFields(ctx, zap.String("trace_id", traceID))
}

// WithSpanID adiciona span_id ao logger do contexto
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return WithFields(ctx, zap.String("span_id", spanID))
}

// WithTenant adiciona tenant_id ao logger do contexto
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return WithFields(ctx, zap.String("tenant_id", tenantID))
}

// WithCliente adiciona cliente_id ao logger do contexto
func WithCliente(ctx context.Context, clienteID string) context.Context {
	return WithFields(ctx, zap.String("cliente_id", clienteID))
}

// Debug logs em nível DEBUG com o logger do contexto
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Debug(msg, fields...)
}

// Info logs em nível INFO com o logger do contexto
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Info(msg, fields...)
}

// Warn logs em nível WARN com o logger do contexto
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Warn(msg, fields...)
}

// Error logs em nível ERROR com o logger do contexto
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Error(msg, fields...)
}

// Fatal logs em nível FATAL com o logger do contexto
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Fatal(msg, fields...)
}

// NewContext cria um novo contexto com trace_id e span_id
func NewContext(ctx context.Context, traceID, spanID string) context.Context {
	ctx = WithTraceID(ctx, traceID)
	ctx = WithSpanID(ctx, spanID)
	return ctx
}

// GenerateTraceID gera um novo trace_id
func GenerateTraceID() string {
	// Implementação simples - em produção usar UUID ou OpenTelemetry
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// GenerateSpanID gera um novo span_id
func GenerateSpanID() string {
	return randomString(6)
}

// randomString gera uma string aleatória
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
