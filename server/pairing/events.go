package pairing

// EventType type for event types.
type EventType string

const (
	// Both Sender and Receiver

	EventConnectionError   EventType = "connection-error"
	EventConnectionSuccess EventType = "connection-success"
	EventTransferError     EventType = "transfer-error"
	EventTransferSuccess   EventType = "transfer-success"

	// Only Receiver side

	EventReceivedAccount EventType = "received-account"
	EventProcessSuccess  EventType = "process-success"
	EventProcessError    EventType = "process-error"
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
)
