package peers

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/signal"
)

// SendDiscoverySummary sends discovery.summary signal.
func SendDiscoverySummary(peers []*p2p.PeerInfo) {
	summary := map[string]int{}
	for i := range peers {
		for _, cap := range peers[i].Caps {
			summary[cap]++
		}
	}
	signal.SendDiscoverySummary(summary)
}
