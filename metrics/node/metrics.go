package node

import (
	"errors"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
)

var (
	nodePeersCounter  metrics.Counter
	nodePeersGauge    metrics.Gauge
	nodeMaxPeersGauge metrics.Gauge
)

func init() {
	nodePeersCounter = metrics.NewRegisteredCounter("p2p/Peers", nil)
	nodePeersGauge = metrics.NewRegisteredGauge("p2p/PeersAbsolute", nil)
	nodeMaxPeersGauge = metrics.NewRegisteredGauge("p2p/MaxPeers", nil)
}

func updateNodeMetrics(node *node.Node, evType p2p.PeerEventType) error {
	change, err := computeMetrics(node, evType)
	if err != nil {
		return err
	}

	// Counter is 1 or -1 based on evType
	nodePeersCounter.Inc(change.Counter)

	nodePeersGauge.Update(change.Absolute)
	nodeMaxPeersGauge.Update(change.Max)

	return nil
}

type peersChange struct {
	Counter  int64
	Absolute int64
	Max      int64
}

func computeMetrics(node *node.Node, evType p2p.PeerEventType) (result peersChange, err error) {
	server := node.Server()
	if server == nil {
		return result, errors.New("p2p server is unavailable")
	}

	if evType == p2p.PeerEventTypeAdd {
		result.Counter = 1
	} else if evType == p2p.PeerEventTypeDrop {
		result.Counter = -1
	}

	result.Absolute = int64(server.PeerCount())
	result.Max = int64(server.MaxPeers)
	return
}
