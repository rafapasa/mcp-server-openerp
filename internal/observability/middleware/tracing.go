package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/rafapasa/mcp-server-openerp/internal/observability/tracing"
)

func TracingFiber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Pega propagator e extrai contexto
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.UserContext(), propagation.HeaderCarrier(c.GetReqHeaders()))

		tracer := tracing.GetTracerProvider().Tracer("mcp-webhook-fiber")

		spanName := c.Method() + " " + c.Path()
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Method()),
				semconv.HTTPRoute(c.Path()),
				semconv.HTTPURL(c.OriginalURL()),
				semconv.HTTPClientIP(c.IP()),
			),
		)
		defer span.End()

		// Injeta contexto com span pro handler seguinte
		c.SetUserContext(ctx)

		err := c.Next()

		span.SetAttributes(semconv.HTTPStatusCode(c.Response().StatusCode()))

		return err
	}
}
