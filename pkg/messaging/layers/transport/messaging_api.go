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
	// Post posts a message on the messaging network. The payload is encoded
	// for WakuMessage version=1 inside the implementation.
	//
	// Deprecated: callers should encode via encoding.EncodeV1 and publish
	// the result through Send. Post is kept transitionally until the
	// transport's send paths are routed through Send (#7462).
	Post(ctx context.Context, req types.NewMessage) ([]byte, error)

	// Send publishes a pre-encoded payload on the messaging network. The
	// payload is expected to be already encoded for WakuMessage version=1
	// (see encoding.EncodeV1). Returns the wire hash on success.
	Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) ([]byte, error)

	// GetFilterMessages returns the messages that match the filter criteria
	// and were received between the last poll and now.
	GetFilterMessages(id string) ([]*types.Message, error)
}

// Compile-time assertion: the waku adapter satisfies MessagingAPI.
var _ MessagingAPI = (*wakuv2.Waku)(nil)
