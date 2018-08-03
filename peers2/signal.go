package peers2

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/signal"
)

type summary struct {
	*p2p.PeerInfo
	Topics []discv5.Topic `json:"topics"`
}

// sendDiscoverySummary sends discovery.summary signal.
func sendDiscoverySummary(peerInfos []*p2p.PeerInfo, poolPeerInfos map[discover.NodeID]*peerInfo) {
	var summaries []*summary
	for _, peerInfo := range peerInfos {
		s := summary{PeerInfo: peerInfo}

		nodeID, err := discover.HexID(peerInfo.ID)
		if err != nil {
			log.Error("failed to compute NodeID", "ID", peerInfo.ID)
		} else {
			if info, ok := poolPeerInfos[nodeID]; ok {
				s.Topics = info.topics
			}
		}

		summaries = append(summaries, &s)
	}
	signal.SendDiscoverySummary(summaries)
}
