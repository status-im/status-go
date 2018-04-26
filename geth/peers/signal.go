package peers

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/geth/signal"
)

const (
	// DiscoveryStarted is sent when node discv5 was started.
	DiscoveryStarted = "discovery.started"
	// DiscoveryStopped is sent when discv5 server was stopped.
	DiscoveryStopped = "discovery.stopped"

	// DiscoverySummary is sent when peer is added or removed.
	// it will be a map with capability=peer count k/v's.
	DiscoverySummary = "discovery.summary"
)

// SendDiscoveryStarted sends discovery.started signal.
func SendDiscoveryStarted() {
	signal.Send(signal.Envelope{
		Type: DiscoveryStarted,
	})
}

// SendDiscoveryStopped sends discovery.stopped signal.
func SendDiscoveryStopped() {
	signal.Send(signal.Envelope{
		Type: DiscoveryStopped,
	})
}

// SendDiscoverySummary sends discovery.summary signal.
func SendDiscoverySummary(peers []*p2p.PeerInfo) {
	summary := map[string]int{}
	for i := range peers {
		for _, cap := range peers[i].Caps {
			summary[cap]++
		}
	}
	signal.Send(signal.Envelope{
		Type:  DiscoverySummary,
		Event: summary,
	})
}
