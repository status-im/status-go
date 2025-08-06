package events

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/protocol/encryption"
	wakutypes "github.com/status-im/status-go/waku/types"
)

// SentMessage reprent a message that has been passed to the transport layer
type SentMessage struct {
	PublicKey  *ecdsa.PublicKey
	Spec       *encryption.ProtocolMessageSpec
	MessageIDs [][]byte
}

type MessageEventType uint32

const (
	MessageScheduled MessageEventType = iota + 1
	MessageSent
	RawMessageSent
)

type MessageEvent struct {
	Recipient   *ecdsa.PublicKey
	Type        MessageEventType
	SentMessage *SentMessage
	RawMessage  *types.RawMessage
}

type DatasyncMessagesSentEvent struct {
	Messages []*wakutypes.NewMessage
}
