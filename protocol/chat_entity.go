package protocol

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/protocol/protobuf"
)

type ChatEntity interface {
	GetChatId() string
	GetMessageType() protobuf.MessageType
	GetSigPubKey() *ecdsa.PublicKey
}
