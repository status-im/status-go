// Package filtertrace defines optional, identifier-only Filter observations.
package filtertrace

import (
	"strconv"
	"sync/atomic"
)

// Observer must return promptly. No message bodies are exposed. Callbacks may
// run concurrently and, for queue events, while a subscription lock is held.
type Observer func(Event)

type Event struct {
	Stage, LocalPeerID, RemotePeerID, TraceID, StreamID     string
	EnvelopeHash, PubsubTopic, ContentTopic, SubscriptionID string
	Reason, Error                                           string
	MatchingPeers                                           []string
	TimestampNS, DurationNS                                 int64
	Rate                                                    float64
	Burst, QueueDepth, QueueCapacity                        int
}

var ids atomic.Uint64

// NewID is process-local; it is not a shared ID between endpoints.
func NewID() string { return strconv.FormatUint(ids.Add(1), 10) }
