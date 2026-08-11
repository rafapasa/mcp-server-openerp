// internal/observability/tracing/tracer.go
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer trace.Tracer
	tp     *sdktrace.TracerProvider
)

// Config contém as configurações do tracer
type Config struct {
	Enabled      bool    `mapstructure:"TRACING_ENABLED"`
	Endpoint     string  `mapstructure:"TRACING_ENDPOINT"`
	ServiceName  string  `mapstructure:"TRACING_SERVICE_NAME"`
	SamplingRate float64 `mapstructure:"TRACING_SAMPLING_RATE"`
}

// DefaultConfig retorna a configuração padrão
func DefaultConfig() Config {
	return Config{
		Enabled:      false,
		Endpoint:     "localhost:4317",
		ServiceName:  "mcp-server",
		SamplingRate: 0.1, // 10% das requisições
	}
}

// Init inicializa o tracer
func Init(cfg Config) error {
	if !cfg.Enabled {
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		return nil
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = "mcp-server"
	}

	// Cria o resource com informações do serviço
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
		),
		resource.WithProcessPID(),
		resource.WithHost(),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Cria o exporter OTLP (gRPC)
	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// Define a sampling rate
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(cfg.SamplingRate),
	)

	// Cria o TracerProvider
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Define o global TracerProvider
	otel.SetTracerProvider(tp)

	// Define o propagador (W3C TraceContext + Baggage)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Cria o tracer
	tracer = tp.Tracer("github.com/rafapasa/mcp-server-openerp")

	return nil
}

// GetTracer retorna o tracer global
func GetTracer() trace.Tracer {
	if tracer == nil {
		return otel.Tracer("default")
	}
	return tracer
}

// GetTracerProvider retorna o TracerProvider global
func GetTracerProvider() *sdktrace.TracerProvider {
	return tp
}

// Shutdown finaliza o tracer
func Shutdown(ctx context.Context) error {
	if tp != nil {
		return tp.Shutdown(ctx)
	}
	return nil
}

// StartSpan inicia um novo span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := GetTracer()
	return tracer.Start(ctx, name, opts...)
}

// SpanFromContext retorna o span atual do contexto
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TraceIDFromContext retorna o trace_id do contexto
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// SpanIDFromContext retorna o span_id do contexto
func SpanIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// AddEvent adiciona um evento ao span atual
func AddEvent(ctx context.Context, name string, attrs ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.AddEvent(name, attrs...)
	}
}

// RecordError registra um erro no span atual
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.RecordError(err, opts...)
	}
}
