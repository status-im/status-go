package pairing

import "github.com/status-im/status-go/multiaccounts"

// EventType type for event types.
type EventType string

const (
	// Both Sender and Receiver

	EventPeerDiscovered       EventType = "peer-discovered"
	EventConnectionError      EventType = "connection-error"
	EventConnectionSuccess    EventType = "connection-success"
	EventTransferError        EventType = "transfer-error"
	EventTransferSuccess      EventType = "transfer-success"
	EventReceivedInstallation EventType = "received-installation"

	// Only Receiver side

	EventReceivedAccount       EventType = "received-account"
	EventProcessSuccess        EventType = "process-success"
	EventProcessError          EventType = "process-error"
	EventReceivedKeystoreFiles EventType = "received-keystore-files"
)

// Event is a type for transfer events.
type Event struct {
	Type   EventType `json:"type"`
	Error  string    `json:"error,omitempty"`
	Action Action    `json:"action"`
	Data   any       `json:"data,omitempty"`
}

type Action int

const (
	ActionConnect Action = iota + 1
	ActionPairingAccount
	ActionSyncDevice
	ActionPairingInstallation
	ActionPeerDiscovery
	ActionKeystoreFilesTransfer
)

type AccountData struct {
	Account  *multiaccounts.Account `json:"account,omitempty"`
	Password string                 `json:"password,omitempty"`
	ChatKey  string                 `json:"chatKey,omitempty"`
}
