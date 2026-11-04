package waku

import (
	"github.com/waku-org/go-waku/waku/v2/protocol/filtertrace"

	"github.com/status-im/status-go/pkg/messaging/diagnostics"
)

func filterTraceObserver() filtertrace.Observer {
	if !diagnostics.Enabled() {
		return nil
	}
	return func(e filtertrace.Event) {
		event := diagnostics.Event{Stage: e.Stage, Source: "filter", TimestampNS: e.TimestampNS,
			LocalPeerID: e.LocalPeerID, RemotePeerID: e.RemotePeerID, FilterTraceID: e.TraceID, StreamID: e.StreamID,
			PubsubTopic: e.PubsubTopic, ContentTopic: e.ContentTopic, SubscriptionID: e.SubscriptionID,
			Reason: e.Reason, Error: e.Error, MatchingPeers: e.MatchingPeers,
			Rate: e.Rate, Burst: e.Burst, DurationNS: e.DurationNS, QueueDepth: e.QueueDepth, QueueCapacity: e.QueueCapacity}
		if e.EnvelopeHash != "" {
			event.EnvelopeHashes = []string{e.EnvelopeHash}
		}
		diagnostics.Emit(event)
	}
}
