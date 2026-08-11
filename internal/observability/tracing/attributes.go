// internal/observability/tracing/attributes.go
package tracing

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Atributos comuns usados nos spans
const (
	// Atributos de mensagem
	AttrMessageID   = attribute.Key("message.id")
	AttrMessageText = attribute.Key("message.text")
	AttrMessageFrom = attribute.Key("message.from")
	AttrMessageType = attribute.Key("message.type")
	AttrMessageSize = attribute.Key("message.size")

	// Atributos de cliente/tenant
	AttrTenantID   = attribute.Key("tenant.id")
	AttrClientID   = attribute.Key("client.id")
	AttrClientName = attribute.Key("client.name")

	// Atributos de LLM
	AttrLLMProvider = attribute.Key("llm.provider")
	AttrLLMModel    = attribute.Key("llm.model")
	AttrLLMTokens   = attribute.Key("llm.tokens")
	AttrLLMPrompt   = attribute.Key("llm.prompt")
	AttrLLMResponse = attribute.Key("llm.response")
	AttrLLMIntent   = attribute.Key("llm.intent")

	// Atributos de banco de dados
	AttrDBTable     = attribute.Key("db.table")
	AttrDBOperation = attribute.Key("db.operation")
	AttrDBQuery     = attribute.Key("db.query")
	AttrDBRows      = attribute.Key("db.rows")

	// Atributos de carrinho
	AttrCartItems  = attribute.Key("cart.items")
	AttrCartTotal  = attribute.Key("cart.total")
	AttrCartAction = attribute.Key("cart.action")

	// Atributos de erro
	AttrErrorType = attribute.Key("error.type")
)

// WithTraceID adiciona trace_id ao span
func WithTraceID(traceID string) trace.SpanStartOption {
	return trace.WithAttributes(
		attribute.String("trace_id", traceID),
	)
}

// WithTenant adiciona tenant_id ao span
func WithTenant(tenantID string) trace.SpanStartOption {
	return trace.WithAttributes(
		AttrTenantID.String(tenantID),
	)
}

// WithClient adiciona cliente_id ao span
func WithClient(clientID string) trace.SpanStartOption {
	return trace.WithAttributes(
		AttrClientID.String(clientID),
	)
}

// WithLLM adiciona atributos de LLM
func WithLLM(provider, model string) trace.SpanStartOption {
	return trace.WithAttributes(
		AttrLLMProvider.String(provider),
		AttrLLMModel.String(model),
	)
}
