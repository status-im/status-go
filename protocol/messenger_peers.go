package protocol

import (
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func (m *Messenger) Peers() messagingtypes.PeerStats {
	return m.messaging.Peers()
}
