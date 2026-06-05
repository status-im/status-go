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

	// SubscribeMessageReceived returns a reliable channel delivering every
	// message received from the network as a neutral ReceivedMessage. The
	// transport decodes, routes by content topic, and matches against its own
	// filters. Call UnsubscribeMessageReceived when done.
	SubscribeMessageReceived() chan *types.ReceivedMessage
	UnsubscribeMessageReceived(ch chan *types.ReceivedMessage)
}

// Compile-time assertion: the waku adapter satisfies MessagingAPI.
var _ MessagingAPI = (*wakuv2.Waku)(nil)
