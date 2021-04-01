package wallet

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// EventType type for event types.
type EventType string

const (
	// EventNewTransfers emitted when new block was added to the same canonical chan.
	EventNewTransfers EventType = "new-transfers"
	// EventFetchingRecentHistory emitted when fetching of lastest tx history is started
	EventFetchingRecentHistory EventType = "recent-history-fetching"
	// EventRecentHistoryReady emitted when fetching of lastest tx history is started
	EventRecentHistoryReady EventType = "recent-history-ready"
	// EventFetchingHistoryError emitted when fetching of tx history failed
	EventFetchingHistoryError EventType = "fetching-history-error"
	// EventNonArchivalNodeDetected emitted when a connection to a non archival node is detected
	EventNonArchivalNodeDetected EventType = "non-archival-node-detected"
)

// Event is a type for wallet events.
type Event struct {
	Type        EventType        `json:"type"`
	BlockNumber *big.Int         `json:"blockNumber"`
	Accounts    []common.Address `json:"accounts"`
	Message     string           `json:"message"`
}
