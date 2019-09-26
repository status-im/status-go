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

	change, err := computeMetrics(n, p2p.PeerEventTypeAdd)
	require.NoError(t, err)
	require.Equal(t, int64(1), change.Counter)
	require.Equal(t, int64(10), change.Max)

	// skip other events
	change, err = computeMetrics(n, p2p.PeerEventTypeMsgRecv)
	require.NoError(t, err)
	require.Equal(t, int64(0), change.Counter)
	change, err = computeMetrics(n, p2p.PeerEventTypeMsgSend)
	require.NoError(t, err)
	require.Equal(t, int64(0), change.Counter)

	change, err = computeMetrics(n, p2p.PeerEventTypeDrop)
	require.NoError(t, err)
	require.Equal(t, int64(-1), change.Counter)

	n.Server().MaxPeers = 20
	change, err = computeMetrics(n, p2p.PeerEventTypeDrop)
	require.NoError(t, err)
	require.Equal(t, int64(20), change.Max)
}
