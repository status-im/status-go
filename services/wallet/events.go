package wallet

import "math/big"

type EventType string

const (
	// EventNewBlock emited when new block was added to the same canonical chan.
	EventNewBlock = "newblock"
	// EventReorg emited when canonical chain was changed. In this case BlockNumber will be an earliest added block.
	EventReorg = "reorg"
)

type Event struct {
	Type        EventType
	BlockNumber *big.Int
}
