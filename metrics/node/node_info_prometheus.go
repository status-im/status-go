// +build metrics,prometheus

package node

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/status-im/status-go/geth/log"
)

var (
	nodePeersGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_peers",
			Help: "A number of node peers",
		},
		[]string{},
	)
)

func init() {
	prometheus.MustRegister(nodePeersGauge)
}

func updateNodeInfo(node *node.Node) {}

func updateNodePeers(node *node.Node) {
	server := node.Server()
	if server == nil {
		log.Warn("Failed to get a server")
		return
	}

	peerCount := server.PeerCount()
	nodePeersGauge.WithLabelValues().Set(float64(peerCount))
}
