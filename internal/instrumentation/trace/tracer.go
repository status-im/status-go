package trace

import (
	oteltrace "go.opentelemetry.io/otel/trace"
	otelnoop "go.opentelemetry.io/otel/trace/noop"
)

// Tracer extends the standard OpenTelemetry Tracer with an Enabled method.
type Tracer interface {
	oteltrace.Tracer
	Enabled() bool
}

func NewTracer(t oteltrace.Tracer) Tracer {
	return &tracer{
		Tracer:  t,
		enabled: true,
	}
}

func NewNoopTracer() Tracer {
	return &tracer{
		Tracer:  otelnoop.NewTracerProvider().Tracer(""),
		enabled: false,
	}
}

type tracer struct {
	oteltrace.Tracer
	enabled bool
}

// This allows callers to check if tracing is enabled before performing expensive operations
// to compute span attributes.
func (t *tracer) Enabled() bool {
	return t.enabled
}
