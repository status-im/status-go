package transport

import (
	"context"

	wakuv2 "github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// MessagingAPI is the transport-layer requirement for sending messages and
// reading received envelopes from the underlying messaging backend.
//
// Currently satisfied by wakuv2.PublicWakuAPI. The logos-delivery Messaging
// API will satisfy it next, after which PublicWakuAPI is retired (pm#380).
type MessagingAPI interface {
	// Post posts a message on the messaging network.
	// Returns the hash of the message on success.
	Post(ctx context.Context, req types.NewMessage) ([]byte, error)

	// GetFilterMessages returns the messages that match the filter criteria
	// and were received between the last poll and now.
	GetFilterMessages(id string) ([]*types.Message, error)
}

// Compile-time assertion: the waku adapter satisfies MessagingAPI.
var _ MessagingAPI = (*wakuv2.PublicWakuAPI)(nil)
