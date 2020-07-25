package protocol

import (
	"crypto/ecdsa"

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
// this function is also required to implement the ChatEntity interface
func (e EmojiReaction) GetSigPubKey() *ecdsa.PublicKey {
	return e.SigPubKey
}
