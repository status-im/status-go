package node

import (
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
)

var (
	nodePeersCounter  = metrics.NewRegisteredCounter("p2p/Peers", nil)
	nodeMaxPeersGauge = metrics.NewRegisteredGauge("p2p/MaxPeers", nil)
)

func updateNodeMetrics(node *node.Node, evType p2p.PeerEventType) {
	server := node.Server()
	if server == nil {
		logger.Error("server not available")
		return
	}

	if evType == p2p.PeerEventTypeAdd {
		nodePeersCounter.Inc(1)
	} else {
		nodePeersCounter.Dec(1)
	}

	nodeMaxPeersGauge.Update(int64(server.MaxPeers))
}
