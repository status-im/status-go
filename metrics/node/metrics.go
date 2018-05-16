package node

import (
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
)

var (
	nodePeersGauge    = metrics.NewRegisteredGauge("p2p/Peers", nil)
	nodeMaxPeersGauge = metrics.NewRegisteredGauge("p2p/MaxPeers", nil)
)

func updateNodeMetrics(node *node.Node) {
	server := node.Server()
	if server == nil {
		logger.Error("server not available")
		return
	}

	nodePeersGauge.Update(int64(server.PeerCount()))
	nodeMaxPeersGauge.Update(int64(server.MaxPeers))
}
