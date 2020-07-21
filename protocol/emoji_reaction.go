package protocol

import "github.com/status-im/status-go/protocol/protobuf"

// EmojiReaction represents an emoji reaction from a user in the application layer, used for persistence, querying and
// signaling
type EmojiReaction struct {
	// ID calculated as keccak256(compressedAuthorPubKey, data) where data is unencrypted payload.
	ID string

	Clock uint64

	// MessageID the ID of the target message that the user wishes to react to
	MessageID string

	// ChatID the ID of the chat the message belongs to, for query efficiency the ChatID is stored in the db even though the
	// target message also stores the ChatID
	ChatID string

	// EmojiID the ID of the emoji the user wishes to react with
	EmojiID protobuf.EmojiReaction_Type

	// From is a public key of the author of the emoji reaction.
	From string

	// Retracted represents whether the user has chosen to remove a previously given reaction
	Retracted bool
}
