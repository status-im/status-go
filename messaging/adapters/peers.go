package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/waku/types"
)

func FromWakuPeerStats(wakuPeers wakutypes.PeerStats) types.PeerStats {
	peerStats := make(types.PeerStats)
	for id, peer := range wakuPeers {
		peerStats[id] = types.Peer{
			Protocols: peer.Protocols,
			Addresses: peer.Addresses,
		}
	}
	return peerStats
}
