package local_pairing

// EventType type for event types.
type EventType string

const (
	EventConnectionError EventType = "connection-error"

	EventConnectionSuccess EventType = "connection-success"

	EventTransferError EventType = "transfer-error"

	EventTransferSuccess EventType = "transfer-success"

	EventDecryptionError EventType = "decryption-error"

	EventInstallationError EventType = "installation-error"

	EventSuccess EventType = "success"
)

// Event is a type for transfer events.
type Event struct {
	Type EventType `json:"type"`
}
