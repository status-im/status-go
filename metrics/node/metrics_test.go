package node

import (
	"testing"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/stretchr/testify/require"
)

func TestUpdateNodeMetricsPeersCounter(t *testing.T) {
	var err error

	n, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers: 10,
		},
		NoUSB: true,
	})
	require.NoError(t, err)
	require.NoError(t, n.Start())
	defer func() { require.NoError(t, n.Stop()) }()

	err = updateNodeMetrics(n, p2p.PeerEventTypeAdd)
	require.NoError(t, err)
	require.Equal(t, int64(1), nodePeersCounter.Count())
	require.Equal(t, int64(10), nodeMaxPeersGauge.Value())

	err = updateNodeMetrics(n, p2p.PeerEventTypeAdd)
	require.NoError(t, err)
	require.Equal(t, int64(2), nodePeersCounter.Count())

	// skip other events
	err = updateNodeMetrics(n, p2p.PeerEventTypeMsgRecv)
	require.NoError(t, err)
	err = updateNodeMetrics(n, p2p.PeerEventTypeMsgSend)
	require.NoError(t, err)
	require.Equal(t, int64(2), nodePeersCounter.Count())

	err = updateNodeMetrics(n, p2p.PeerEventTypeDrop)
	require.NoError(t, err)
	require.Equal(t, int64(1), nodePeersCounter.Count())

	n.Server().MaxPeers = 20
	err = updateNodeMetrics(n, p2p.PeerEventTypeDrop)
	require.NoError(t, err)
	require.Equal(t, int64(20), nodeMaxPeersGauge.Value())
}
