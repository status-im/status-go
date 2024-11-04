package walletconnectevent

import (
	"github.com/status-im/status-go/services/wallet/walletconnect"
)

// EventType type for event types.
type EventType string

// Event is a type for walletConnect events.
type Event struct {
	Type    EventType             `json:"type"`
	Session walletconnect.Session `json:"session"`
}

const (
	// EventTypeAdded is emitted when a new session is added.
	EventTypeAdded EventType = "added"
)
