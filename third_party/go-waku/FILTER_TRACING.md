# Local Filter diagnostics

This directory is a source snapshot of `github.com/waku-org/go-waku v0.10.3`,
used through the version-specific replacement in status-go's go.mod. Upstream
license notices and generated protobuf sources are retained. The snapshot makes
native and Docker builds reproducible without editing the Go module cache.

Local changes add optional identifier-only observer callbacks in:

- `waku/v2/protocol/filtertrace`: callback/event types and process-local IDs.
- `waku/v2/protocol/filter`: relay subscription matching and push results;
  client admission/rejection and validation. Configure with `WithObserver` and
  `WithLightNodeObserver`. A nil callback disables observations.
- `waku/v2/protocol/subscription`: matching subscription queue acceptance/drop.
- `waku/v2/api/filter`: subscription-to-API multiplex queue handoff.

Callbacks must be quick and safe for concurrent calls. Some queue observations
occur while the existing subscription read lock is held. status-go connects them
to its bounded non-blocking `message.trace` emitter when tracing is enabled.
No payload bodies are exposed, and limit, subscription, queue, timeout, wire
format, error-return, and retry behavior are unchanged. In particular, the
existing push writer's nil error return after a write failure is preserved;
the observer reports the actual write failure independently.

`filter_push_written` means the relay wrote the stream message; it does not
confirm client admission. An early client rejection is observed before decoding
and has no envelope hash. Trace IDs and stream IDs are local to one endpoint;
peer/time matches between early rejections and relay attempts are candidates,
not exact per-envelope proof. Keep that distinction in downstream reports.
