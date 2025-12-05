package trace

import (
	"context"
	"crypto/sha256"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Config describes the tracing provider configuration.
type Config struct {
	ServiceName  string
	OTLPEndpoint string
	Insecure     bool
}

// InitProvider configures a global tracer provider for the process.
func InitProvider(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	clientOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.Insecure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(clientOpts...))
	if err != nil {
		return nil, fmt.Errorf("failed to build otlp exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

// DeriveRemoteContext applies DeriveSpanContext to a background context, marking the span as remote.
// Use it when instantiating root spans from external inputs that were converted via DeriveSpanContext.
func DeriveRemoteContext(data []byte) context.Context {
	return oteltrace.ContextWithRemoteSpanContext(
		context.Background(), DeriveSpanContext(data, true),
	)
}

// DeriveSpanContext builds a sampled span context from provided data,
// making it possible to recreate trace/span IDs from stable identifiers (e.g. message IDs)
// whenever standard OTEL header propagation is unavailable or undesirable.
func DeriveSpanContext(data []byte, remote bool) oteltrace.SpanContext {
	traceID, spanID := deriveTraceAndSpanID(data)
	return oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     remote,
	})
}

func deriveTraceAndSpanID(data []byte) (oteltrace.TraceID, oteltrace.SpanID) {
	var traceID oteltrace.TraceID
	var spanID oteltrace.SpanID

	if len(data) == 0 {
		return traceID, spanID
	}

	sum := sha256.Sum256(data)
	copy(traceID[:], sum[:16])
	copy(spanID[:], sum[16:24])
	return traceID, spanID
}
