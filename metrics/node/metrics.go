package node

import (
	"errors"
	"flag"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
)

var (
	nodePeersCounter  metrics.Counter
	nodeMaxPeersGauge metrics.Gauge
)

func init() {
	// When running tests, we want metrics to be enabled.
	// Having init() in metrics_test.go does not work because
	// this init() is executed first.
	if flag.Lookup("test.v") != nil {
		metrics.Enabled = true
	}

	nodePeersCounter = metrics.NewRegisteredCounter("p2p/Peers", nil)
	nodeMaxPeersGauge = metrics.NewRegisteredGauge("p2p/MaxPeers", nil)
}

func updateNodeMetrics(node *node.Node, evType p2p.PeerEventType) error {
	server := node.Server()
	if server == nil {
		return errors.New("p2p server is unavailable")
	}

	if evType == p2p.PeerEventTypeAdd {
		nodePeersCounter.Inc(1)
	} else if evType == p2p.PeerEventTypeDrop {
		nodePeersCounter.Dec(1)
	}

	nodeMaxPeersGauge.Update(int64(server.MaxPeers))

	return nil
}
