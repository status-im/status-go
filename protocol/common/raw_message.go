package common

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/protocol/protobuf"
)

// RawMessage represent a sent or received message, kept for being able
// to re-send/propagate
type RawMessage struct {
	ID                   string
	LocalChatID          string
	LastSent             uint64
	SendCount            int
	Sent                 bool
	ResendAutomatically  bool
	SkipEncryption       bool
	SendPushNotification bool
	MessageType          protobuf.ApplicationMetadataMessage_Type
	Payload              []byte
	Sender               *ecdsa.PrivateKey
	Recipients           []*ecdsa.PublicKey
	DataSyncID           []byte
	SkipGroupMessageWrap bool
	SendOnPersonalTopic  bool
}
