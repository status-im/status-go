package types

import "time"

// ConnectionState mirrors the logos-delivery Messaging API's three-state
// connection status (see logos-delivery waku/api/types.nim `ConnectionStatus`).
// It lets status-go speak the same vocabulary as the Messaging API:
//
//   - Disconnected:        no connectivity.
//   - PartiallyConnected:  some peers, but the mesh is not yet healthy.
//   - Connected:           a healthy mesh.
//
// Online detection derives from it: a node is online whenever it is not
// Disconnected (both PartiallyConnected and Connected count as online), matching
// logos-delivery's `status != Disconnected` semantics.
type ConnectionState int

const (
	ConnectionStateDisconnected ConnectionState = iota
	ConnectionStatePartiallyConnected
	ConnectionStateConnected
)

// IsOnline reports whether the node has any connectivity.
func (s ConnectionState) IsOnline() bool {
	return s != ConnectionStateDisconnected
}

// String returns a stable, host-facing label (also used as the signal payload).
func (s ConnectionState) String() string {
	switch s {
	case ConnectionStatePartiallyConnected:
		return "partiallyConnected"
	case ConnectionStateConnected:
		return "connected"
	default:
		return "disconnected"
	}
}

// HistoryReconcileWindow is a period during which live delivery could not be
// trusted. Store reconciliation must not query beyond these bounds (apart from
// the caller's small out-of-order tolerance).
type HistoryReconcileWindow struct {
	From time.Time
	To   time.Time
}
