package wallet

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// EventType type for event types.
type EventType string

const (
	// EventNewBlock emitted when new block was added to the same canonical chan.
	EventNewBlock EventType = "newblock"
	// EventReorg emitted when canonical chain was changed. In this case, BlockNumber will be an earliest added block.
	EventReorg EventType = "reorg"
	// EventNewHistory emitted if transfer from older block was added.
	EventNewHistory EventType = "history"
)

// Event is a type for wallet events.
type Event struct {
	Type        EventType        `json:"type"`
	BlockNumber *big.Int         `json:"blockNumber"`
	Accounts    []common.Address `json:"accounts"`
}
