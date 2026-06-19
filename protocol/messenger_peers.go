package protocol

import (
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

// Peers is retained only for the Python functional tests (see tests-functional);
// it is not used by status-app.
func (m *Messenger) Peers() messagingtypes.PeerStats {
	return m.messaging.Peers()
}
