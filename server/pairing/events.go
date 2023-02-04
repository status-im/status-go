package pairing

// EventType type for event types.
type EventType string

const (

	// both client and server
	EventConnectionError EventType = "connection-error"
	EventConnectionSuccess EventType = "connection-success"
	EventTransferError EventType = "transfer-error"
	EventTransferSuccess EventType = "transfer-success"

	// Only receiver side
	EventProcessSuccess EventType = "process-success"
	EventProcessError EventType = "process-error"
)

// Event is a type for transfer events.
type Event struct {
	Type   EventType `json:"type"`
	Error  string    `json:"error,omitempty"`
	Action Action    `json:"action"`
}

type Action int

const (
	ActionConnect = iota + 1
	ActionPairingAccount
	ActionSyncDevice
)
