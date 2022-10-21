package server

// EventType type for event types.
type EventType string

const (
	EventConnectionError EventType = "connection-error"

	EventConnectionSuccess EventType = "connection-success"

	EventTransferError EventType = "transfer-error"

	EventTransferSuccess EventType = "transfer-success"

	EventProcessSuccess EventType = "process-success"

	EventProcessError EventType = "process-error"
)

// Event is a type for transfer events.
type Event struct {
	Type  EventType `json:"type"`
	Error string    `json:"error,omitempty"`
}
