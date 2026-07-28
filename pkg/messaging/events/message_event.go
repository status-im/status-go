package events

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/pkg/messaging/types"
)

// SentMessage represents a message that has been passed to the transport layer
type SentMessage struct {
	PublicKey     *ecdsa.PublicKey
	Installations []*types.Installation
	MessageIDs    [][]byte
}

// DeliveredMessage identifies application messages confirmed delivered by a
// reliability layer. Community SDS confirmations are translated to application
// message IDs before this event is published.
type DeliveredMessage struct {
	MessageIDs [][]byte
}
