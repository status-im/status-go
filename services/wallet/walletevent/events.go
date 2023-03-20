package walletevent

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// EventType type for event types.
type EventType string

// Event is a type for transfer events.
type Event struct {
	Type        EventType        `json:"type"`
	BlockNumber *big.Int         `json:"blockNumber"`
	Accounts    []common.Address `json:"accounts"`
	Message     string           `json:"message"`
	At          int64            `json:"at"`
	ChainID     uint64           `json:"chainId"`
}
