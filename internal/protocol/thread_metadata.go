package protocol

import (
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/internal/protocol/protobuf"
)

type ThreadMetadata struct {
	*protobuf.ThreadMetadata
	SigPubKey *ecdsa.PublicKey
}

func (m *ThreadMetadata) GetSigPubKey() *ecdsa.PublicKey {
	return m.SigPubKey
}

func (m *ThreadMetadata) GetProtobuf() proto.Message {
	return m.ThreadMetadata
}

func (m *ThreadMetadata) SetMessageType(messageType protobuf.MessageType) {
	m.MessageType = messageType
}

func (m *ThreadMetadata) WrapGroupMessage() bool {
	return false
}
