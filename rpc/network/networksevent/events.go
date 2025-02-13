package networksevent

// EventType type for event types.
type EventType string

// Event is a type for networks events.
type Event struct {
	Type EventType `json:"type"`
}

const (
	// EventTypeActiveNetworksChanged is emitted when networks are activated/deactivated. This includes Testnet mode change.
	EventTypeActiveNetworksChanged EventType = "active-networks-changed"
)
