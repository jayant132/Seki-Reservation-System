// Package tracing wires up OpenTelemetry distributed tracing. Combined
// with the existing structured logs and Prometheus metrics, this gives
// the three standard observability pillars: every request gets a trace ID
// that shows up in both the logs and the response header, and can be
// followed end-to-end (HTTP handler -> service -> the exact SQL
// transaction) in the bundled Jaeger UI. For a system whose entire
// purpose is proving correctness under concurrency, being able to see
// each request's timeline individually during a load test is directly
// useful, not just decorative.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Init configures the global tracer provider to export spans to an OTLP
// collector (Jaeger's OTLP-gRPC receiver in this project's Docker Compose
// stack) and returns a shutdown function the caller must run before the
// process exits, so buffered spans are flushed instead of dropped.
func Init(ctx context.Context, serviceName, otlpEndpoint string) (func(context.Context) error, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otlpEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// Tracer returns the named tracer used to start spans throughout the
// application (handlers, repository transactions, etc).
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// TraceIDFromContext extracts the current span's trace ID as a string, or
// "" if there is no active span. Used to attach the trace ID to log lines
// and to the X-Trace-Id response header so it can be correlated with
// Jaeger from the outside.
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().HasTraceID() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// KV is a small convenience wrapper so call sites don't need to import
// the otel attribute package directly for simple cases.
func KV(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}
