package protocol

import (
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/internal/protocol/protobuf"
)

type DeleteThread struct {
	*protobuf.DeleteThread
	SigPubKey *ecdsa.PublicKey
}

func (m *DeleteThread) GetSigPubKey() *ecdsa.PublicKey {
	return m.SigPubKey
}

func (m *DeleteThread) GetProtobuf() proto.Message {
	return m.DeleteThread
}

func (m *DeleteThread) SetMessageType(messageType protobuf.MessageType) {
	m.MessageType = messageType
}

func (m *DeleteThread) WrapGroupMessage() bool {
	return false
}
