// Package diagnostics provides optional, bounded, asynchronous message tracing.
// It deliberately has no dependency on messaging or the public signal bridge.
package diagnostics

import (
	"context"
	"encoding/hex"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Metadata struct {
	AttemptID      string   `json:"attemptId,omitempty"`
	MessageIDs     []string `json:"messageIds,omitempty"`
	MVDSMessageIDs []string `json:"mvdsMessageIds,omitempty"`
	AckIDs         []string `json:"ackIds,omitempty"`
	Peer           string   `json:"peer,omitempty"`
	Recipient      string   `json:"recipient,omitempty"`
	Reliability    string   `json:"reliability,omitempty"`
}

// TimestampNS is captured at the instrumentation point, before asynchronous
// delivery. Encode it as a string so JavaScript clients do not lose precision.
type Event struct {
	Metadata
	Version            int      `json:"version"`
	Stage              string   `json:"stage"`
	TimestampNS        int64    `json:"timestampNs,string"`
	Sequence           uint64   `json:"sequence"`
	Dropped            uint64   `json:"dropped"`
	EnvelopeHashes     []string `json:"envelopeHashes,omitempty"`
	Source             string   `json:"source,omitempty"`
	Error              string   `json:"error,omitempty"`
	DurationNS         int64    `json:"durationNs,omitempty"`
	PublicationAttempt int      `json:"publicationAttempt,omitempty"`
	RetryDelayNS       int64    `json:"retryDelayNs,omitempty"`
	QueueDepth         int      `json:"queueDepth,omitempty"`
	QueueCapacity      int      `json:"queueCapacity,omitempty"`
	LocalPeerID        string   `json:"localPeerId,omitempty"`
	RemotePeerID       string   `json:"remotePeerId,omitempty"`
	FilterTraceID      string   `json:"filterTraceId,omitempty"`
	StreamID           string   `json:"streamId,omitempty"`
	PubsubTopic        string   `json:"pubsubTopic,omitempty"`
	ContentTopic       string   `json:"contentTopic,omitempty"`
	SubscriptionID     string   `json:"subscriptionId,omitempty"`
	Reason             string   `json:"reason,omitempty"`
	MatchingPeers      []string `json:"matchingPeers,omitempty"`
	Rate               float64  `json:"rate,omitempty"`
	Burst              int      `json:"burst,omitempty"`
}

type emitter struct {
	queue    chan Event
	sequence atomic.Uint64
	dropped  atomic.Uint64
}

var active atomic.Pointer[emitter]
var enableOnce sync.Once
var attempts atomic.Uint64

// Enable is called once by the public signal bridge when tracing is opted in.
// A slow consumer cannot block the messaging path; missing events are observable
// through sequence gaps and the cumulative dropped counter on later events.
func Enable(handler func(Event)) {
	if handler == nil {
		return
	}
	enableOnce.Do(func() {
		e := &emitter{queue: make(chan Event, 4096)}
		active.Store(e)
		go func() {
			for event := range e.queue {
				handler(event)
			}
		}()
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				Emit(Event{Stage: "heartbeat"})
			}
		}()
	})
}

func Enabled() bool { return active.Load() != nil }

func Emit(event Event) {
	e := active.Load()
	if e == nil {
		return
	}
	event.Version = 3
	if event.TimestampNS == 0 {
		event.TimestampNS = time.Now().UnixNano()
	}
	event.Sequence = e.sequence.Add(1)
	event.Dropped = e.dropped.Load()
	select {
	case e.queue <- event:
	default:
		e.dropped.Add(1)
	}
}

type contextKey struct{}

func Begin(ctx context.Context, messageIDs [][]byte, recipient, reliability string) (context.Context, Metadata) {
	if !Enabled() {
		return ctx, Metadata{}
	}
	return BeginWithMetadata(ctx, Metadata{MessageIDs: Hexes(messageIDs), Recipient: recipient, Reliability: reliability})
}

// BeginWithMetadata preserves reliability IDs on every subsequent envelope event.
func BeginWithMetadata(ctx context.Context, metadata Metadata) (context.Context, Metadata) {
	if !Enabled() {
		return ctx, Metadata{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	metadata.AttemptID = strconv.FormatInt(time.Now().UnixNano(), 16) + "-" + strconv.FormatUint(attempts.Add(1), 16)
	Emit(Event{Metadata: metadata, Stage: "attempt_started"})
	return context.WithValue(ctx, contextKey{}, metadata), metadata
}

func FromContext(ctx context.Context) Metadata {
	if !Enabled() || ctx == nil {
		return Metadata{}
	}
	metadata, _ := ctx.Value(contextKey{}).(Metadata)
	return metadata
}

func Hexes(values [][]byte) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = "0x" + hex.EncodeToString(value)
	}
	return result
}
