package networksevent

// EventType type for event types.
type EventType string

// Event is a type for networks events.
type Event struct {
	Type                        EventType                    `json:"type"`
	ActiveNetworksChangedParams *ActiveNetworksChangedParams `json:"activeNetworksChangedParams,omitempty"` // Only for EventTypeActiveNetworksChanged
}

const (
	// EventTypeActiveNetworksChanged is emitted when networks are activated/deactivated. This includes Testnet mode change.
	EventTypeActiveNetworksChanged EventType = "active-networks-changed"
)

type ActiveNetworksChangedParams struct {
	ActivatedChainIDs     []uint64 `json:"activatedChainIds"`
	DeactivatedChainIDs   []uint64 `json:"deactivatedChainIds"`
	CurrentActiveChainIDs []uint64 `json:"currentActiveChainIds"`
}
