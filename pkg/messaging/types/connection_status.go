package types

import (
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// ConnectionState is the three-state connection status (Disconnected /
// PartiallyConnected / Connected) surfaced to the Messaging API, re-exported
// from the waku layer that produces it.
type ConnectionState = wakutypes.ConnectionState

const (
	ConnectionStateDisconnected       = wakutypes.ConnectionStateDisconnected
	ConnectionStatePartiallyConnected = wakutypes.ConnectionStatePartiallyConnected
	ConnectionStateConnected          = wakutypes.ConnectionStateConnected
)

type ConnectionStatus struct {
	IsOnline bool            `json:"isOnline"`
	State    ConnectionState `json:"state"`
}

type ConnectionStatusSubscription interface {
	C() <-chan ConnectionStatus
	Unsubscribe()
}
