## Tracing

> Traces give us the big picture of what happens when a request is made to an application. Whether your application is a monolith with a single database or a sophisticated mesh of services, traces are essential to understanding the full “path” a request takes in your application.

https://opentelemetry.io/docs/concepts/signals/traces/

## Local stack

Run the helper compose file to start OpenSearch, OpenSearch Dashboards, and Jaeger:

```bash
cd tools/opentelemetry
docker compose up
```

Services:

- `opensearch` (HTTP API on `localhost:9200`)
- `opensearch-dashboards` (UI on `localhost:5601`)
- `jaeger` (UI on `localhost:16686`)

## Enabling tracing in status-go

Enable tracing by configuring `NodeConfig`:

```go
nodeCfg.OTELConfig = params.OTELConfig{
    Enabled:  true,
    Endpoint: "localhost:4317",
    Insecure: true,
}
```

Integration tests that bypass `GethStatusBackend` can bootstrap tracing on demand:

```go
ctx := context.Background()
shutdownTracer, err := trace.InitProvider(ctx, trace.Config{
    ServiceName:  "status-go-tests",
    OTLPEndpoint: "localhost:4317",
    Insecure:     true,
})
defer shutdownTracer(ctx)
s.Require().NoError(err)
```

## Tracing helpers

The reusable helpers live in `internal/instrumentation/trace`:

- `trace.InitProvider` wires the global OpenTelemetry tracer provider and should be called once per process (see the examples above).
- `trace.Tracer` wraps `otel.Tracer` and exposes `Enabled()` so expensive attribute calculations when tracing is disabled can be conveniently skipped.
- `trace.DeriveSpanContext`/`trace.DeriveRemoteContext` turn any byte slice (for example a message hash) into deterministic trace/span IDs.

## Adding spans in status-go

1. Accept a `trace.Tracer` in each instrumented component (messenger, sender, etc.).
2. Carry `context.Context` through the chain of instrumented functions.
3. Start spans with `ctx, span := tracer.Start(ctx, "Component.operation")` and ensure `span.End()` executes.
4. Enrich spans with attributes/events via `span.SetAttributes` / `span.AddEvent` so searches in Jaeger/OpenSearch remain useful.

Example:

```go
func (m *Messenger) sendChatMessage(ctx context.Context, message *common.Message) (*MessengerResponse, error) {
	ctx, span := m.tracer.Start(ctx, "Messenger.sendChatMessage",
		oteltrace.WithAttributes(
			otelattribute.Stringer("messageType", message.MessageType),
			otelattribute.Stringer("contentType", message.ContentType),
		),
	)
	defer span.End()

    // ... send chat message logic ...

    // Context is passed to another instrumented function.
	rawMessage, err = m.dispatchMessage(ctx, rawMessage)
	if err != nil {
		return nil, err
	}

    // ... send chat message logic ...

    return response, nil
}
```

With this pattern Jaeger shows the full waterfall, while OpenSearch Dashboards can help analyzing spans through `jaeger-span-*` index.

## Messages tracing

### Classical header-based propagation

Distributed traces typically span multiple services running on different machines. The transport propagates context as metadata (for example W3C `traceparent` or Zipkin B3 headers) so visualizations form a single waterfall such as:

```
profileService.UploadAvatar
  └─ mediaService.StoreBlob
      └─ thumbnailService.GeneratePreview
```

### Why status-go is different

Status-go needs to follow the same logical message as it moves between different hosts of the *same service*. Payloads may be segmented or relayed through the reliability layer, so injecting extra headers would either break delivery or clutter the protocol. The ideal single-trace view would be:

```
status-go.Messenger.PublishCommunityDescription (host: owner)
  └─ status-go.Sender.SendPublicMessage (host: owner)
      └─ status-go.Processor.ProcessMessage (host: memberA)
          └─ status-go.Messenger.UpdateCommunity (host: memberA)
      └─ status-go.Processor.ProcessMessage (host: memberB)
          └─ status-go.Messenger.UpdateCommunity (host: memberB)
```

Instead of mutating payloads, the system derives trace context from stable identifiers (message IDs / hashes) and links sender/receiver spans together. That produces two cooperating traces:

**TraceID "abc"** (publisher):

```
status-go.Messenger.PublishCommunityDescription (host: owner)
  └─ status-go.Sender.SendPublicMessage (host: owner, link: "xyz") // link derived from message ID
```

**TraceID "xyz"** (recipients):

```
└─ status-go.Processor.ProcessMessage (host: memberA) // root derived from received message ID
    └─ status-go.Messenger.UpdateCommunity (host: memberA)
└─ status-go.Processor.ProcessMessage (host: memberB)
    └─ status-go.Messenger.UpdateCommunity (host: memberB)
```

### Instrumentation patterns

**Sender side**

```go
linkSpanCtx := trace.DeriveSpanContext([]byte(message.ID), false)

spanOpts := []oteltrace.SpanStartOption{
	oteltrace.WithAttributes(
		otelattribute.String("messageID", message.ID),
		otelattribute.Stringer("messageType", message.MessageType),
	),
	oteltrace.WithLinks(oteltrace.Link{SpanContext: linkSpanCtx}),
}

ctx, span := s.tracer.Start(ctx, "Sender.sendMessage", spanOpts...)
defer span.End()
```

`DeriveSpanContext` supplies a stable context for links that must reference work performed elsewhere.

**Receiver side**

```go
ctx := trace.DeriveRemoteContext(messageID)
ctx, span := p.tracer.Start(ctx, "Processor.processMessage")
defer span.End()
```

`DeriveRemoteContext` recreates the *root* of a trace from external data so each recipient continues the narrative.
