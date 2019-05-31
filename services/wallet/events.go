package wallet

import "math/big"

// EventType type for event types.
type EventType string

const (
	// EventNewBlock emitted when new block was added to the same canonical chan.
	EventNewBlock = "newblock"
	// EventReorg emitted when canonical chain was changed. In this case, BlockNumber will be an earliest added block.
	EventReorg = "reorg"
)

// Event is a type for wallet events.
type Event struct {
	Type        EventType
	BlockNumber *big.Int
}
