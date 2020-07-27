package common

import (
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/protobuf"
)

type ChatEntity interface {
	proto.Message

	GetChatId() string
	GetMessageType() protobuf.MessageType
	GetSigPubKey() *ecdsa.PublicKey
	GetProtobuf() proto.Message

	SetMessageType(messageType protobuf.MessageType)
}
