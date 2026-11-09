package transport

import (
	"context"

	"github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// MessagingAPI is the transport-layer requirement for sending messages and
// reading received envelopes from the underlying messaging backend.
//
// Currently satisfied by *waku.Waku. The logos-delivery Messaging API will
// satisfy it next, after which the waku adapter is retired (pm#380).
type MessagingAPI interface {
	// Send queues a payload already encrypted with rfc26.Encode and returns the
	// ID of the request. The outcome arrives later as an envelope event with
	// that ID: EventEnvelopeSent, which carries the wire hash, or
	// EventEnvelopeExpired.
	Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) (types.RequestID, error)

	// SubscribeEnvelopeEvents returns the backend's event stream. The transport
	// consumes it both for reception (EventEnvelopeAvailable carries a neutral
	// *types.ReceivedMessage in Data, which the transport decodes and routes to
	// its filters) and for the send-side lifecycle events the EnvelopesMonitor
	// tracks. This is the single, logos-delivery-shaped event seam.
	SubscribeEnvelopeEvents(events chan<- types.EnvelopeEvent) types.Subscription

	// Subscribe and Unsubscribe register and remove wire subscriptions for
	// the given content topics on a pubsub topic, keyed by the (pubsubTopic,
	// contentTopic) pairs themselves: the transport's FilterSubscriptions
	// subscribes when the first filter on a pair is installed and
	// unsubscribes when the last one is removed. Any subscription identifier
	// the backend keeps for a pair is its own bookkeeping. Both are
	// idempotent.
	//
	// NOTE: in the waku backend these currently take effect only in
	// light-client mode (the Waku Filter protocol); relay clients receive
	// everything on the pubsub topics they relay regardless of what is
	// subscribed here, so the calls are pure bookkeeping for them. The
	// semantics therefore do not exactly mirror the logos-delivery Messaging
	// API yet — to be aligned in further PRs.
	Subscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error
	Unsubscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error
}

// ContentTopicOnly is implemented by a MessagingAPI that subscribes and
// delivers by content topic alone. The transport then counts wire
// subscriptions per content topic, passes no pubsub topic to Subscribe and
// Unsubscribe, and routes received messages that carry no pubsub topic to the
// filters on any of them.
type ContentTopicOnly interface {
	ContentTopicOnly()
}

// Compile-time assertion: the waku adapter satisfies MessagingAPI.
var _ MessagingAPI = (*waku.Waku)(nil)
