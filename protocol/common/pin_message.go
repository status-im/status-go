package common

import (
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/protobuf"
)

type PinMessage struct {
	protobuf.PinMessage

	// ID calculated as keccak256(compressedAuthorPubKey, data) where data is unencrypted payload.
	ID string `json:"id"`
	// MessageID string `json:"messageID"`
	// WhisperTimestamp is a timestamp of a Whisper envelope.
	WhisperTimestamp uint64 `json:"whisperTimestamp"`
	// From is a public key of the user who pinned the message.
	From string `json:"from"`
	// The chat id to be stored locally
	LocalChatID string           `json:"localChatId"`
	SigPubKey   *ecdsa.PublicKey `json:"-"`
	// Identicon of the author
	Identicon string `json:"identicon"`
	// Random 3 words name
	Alias string `json:"alias"`
}

type PinnedMessage struct {
	Message
	PinnedAt uint64 `json:"pinnedAt"`
}

// WrapGroupMessage indicates whether we should wrap this in membership information
func (m *PinMessage) WrapGroupMessage() bool {
	return false
}

// SetMessageType a setter for the MessageType field
// this function is required to implement the ChatEntity interface
func (m *PinMessage) SetMessageType(messageType protobuf.MessageType) {
	m.MessageType = messageType
}

func (m *PinMessage) GetGrant() []byte {
	return nil
}

// GetProtoBuf returns the struct's embedded protobuf struct
// this function is required to implement the ChatEntity interface
func (m *PinMessage) GetProtobuf() proto.Message {
	return &m.PinMessage
}

// GetSigPubKey returns an ecdsa encoded public key
// this function is required to implement the ChatEntity interface
func (m PinMessage) GetSigPubKey() *ecdsa.PublicKey {
	return m.SigPubKey
}
