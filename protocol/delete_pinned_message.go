package protocol

import (
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/protobuf"
)

// DeletePinnedMessage represents a delete of a message from a user in the application layer, used for persistence, querying and
// signaling
type DeletePinnedMessage struct {
	protobuf.DeletePinnedMessage

	// ID is the ID of the message that has been edited
	ID string `json:"id,omitempty"`

	// From is a public key of the author of the edit reaction.
	From string `json:"from,omitempty"`

	// SigPubKey is the ecdsa encoded public key of the edit author
	SigPubKey *ecdsa.PublicKey `json:"-"`

	// LocalChatID is the chatID of the local chat (one-to-one are not symmetric)
	LocalChatID string `json:"localChatId"`
}

// GetSigPubKey returns an ecdsa encoded public key
// this function is required to implement the ChatEntity interface
func (e DeletePinnedMessage) GetSigPubKey() *ecdsa.PublicKey {
	return e.SigPubKey
}

// GetProtoBuf returns the struct's embedded protobuf struct
// this function is required to implement the ChatEntity interface
func (e DeletePinnedMessage) GetProtobuf() proto.Message {
	return &e.DeletePinnedMessage
}

// SetMessageType a setter for the MessageType field
// this function is required to implement the ChatEntity interface
func (e *DeletePinnedMessage) SetMessageType(messageType protobuf.MessageType) {
	e.MessageType = messageType
}

// WrapGroupMessage indicates whether we should wrap this in membership information
func (e DeletePinnedMessage) WrapGroupMessage() bool {
	return false
}
