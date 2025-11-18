package events

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/messaging/types"
)

// SentMessage represents a message that has been passed to the transport layer
type SentMessage struct {
	PublicKey     *ecdsa.PublicKey
	Installations []*types.Installation
	MessageIDs    [][]byte
}
