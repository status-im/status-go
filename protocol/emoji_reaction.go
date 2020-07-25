package protocol

import (
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/protobuf"
)

// EmojiReaction represents an emoji reaction from a user in the application layer, used for persistence, querying and
// signaling
type EmojiReaction struct {
	protobuf.EmojiReaction

	// ID calculated as keccak256(compressedAuthorPubKey, data) where data is unencrypted payload.
	ID string

	// From is a public key of the author of the emoji reaction.
	From string

	// Retracted represents whether the user has chosen to remove a previously given reaction
	Retracted bool

	// SigPubKey is the ecdsa encoded public key of the emoji reaction author
	SigPubKey *ecdsa.PublicKey `json:"-"`
}

// GetSigPubKey returns an ecdsa encoded public key
// this function is required to implement the ChatEntity interface
func (e EmojiReaction) GetSigPubKey() *ecdsa.PublicKey {
	return e.SigPubKey
}

// GetProtoBuf returns the struct's embedded protobuf struct
// this function is required to implement the ChatEntity interface
func (e EmojiReaction) GetProtobuf() proto.Message {
	return &e.EmojiReaction
}

// SetMessageType a setter for the MessageType field
// this function is required to implement the ChatEntity interface
func (e *EmojiReaction) SetMessageType(messageType protobuf.MessageType) {
	e.MessageType = messageType
}
