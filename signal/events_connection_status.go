package signal

import (
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

// EventConnectionStatusChange is emitted when the waku node's three-state
// connection status changes. It mirrors the logos-delivery Messaging API's
// connection-status-change event (Disconnected / PartiallyConnected /
// Connected), superseding the removed peer-stats signal.
const EventConnectionStatusChange = "waku.connection.status.change"

// ConnectionStatusSignal is the payload of EventConnectionStatusChange.
type ConnectionStatusSignal struct {
	// Status is the connection state as a stable label:
	// "disconnected" | "partiallyConnected" | "connected".
	Status string `json:"status"`
	// IsOnline is a convenience flag (status != disconnected).
	IsOnline bool `json:"isOnline"`
}

// SendConnectionStatusChange emits the connection-status-change signal.
func SendConnectionStatusChange(status messagingtypes.ConnectionStatus) {
	send(EventConnectionStatusChange, ConnectionStatusSignal{
		Status:   status.State.String(),
		IsOnline: status.IsOnline,
	})
}
