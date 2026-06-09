package transport

import (
	"context"

	wakuv2 "github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// MessagingAPI is the transport-layer requirement for sending messages and
// reading received envelopes from the underlying messaging backend.
//
// Currently satisfied by *wakuv2.Waku. The logos-delivery Messaging API will
// satisfy it next, after which the waku adapter is retired (pm#380).
type MessagingAPI interface {
	// Send publishes a pre-encoded payload on the messaging network. The
	// payload is expected to be already encoded for WakuMessage version=1
	// (see rfc26.Encode). Returns the wire hash on success.
	Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) ([]byte, error)

	// SubscribeEnvelopeEvents returns the backend's event stream. The transport
	// consumes it both for reception (EventEnvelopeAvailable carries a neutral
	// *types.ReceivedMessage in Data, which the transport decodes and routes to
	// its filters) and for the send-side lifecycle events the EnvelopesMonitor
	// tracks. This is the single, logos-delivery-shaped event seam.
	SubscribeEnvelopeEvents(events chan<- types.EnvelopeEvent) types.Subscription
}

// Compile-time assertion: the waku adapter satisfies MessagingAPI.
var _ MessagingAPI = (*wakuv2.Waku)(nil)
