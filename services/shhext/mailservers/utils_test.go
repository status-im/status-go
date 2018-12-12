package mailservers

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/require"
)

func TestGetFirstConnected(t *testing.T) {
	numPeers := 3
	nodes := make([]*enode.Node, numPeers)
	peers := make([]*p2p.Peer, numPeers)
	nodesMap := getMapWithRandomNodes(t, numPeers)
	i := 0
	for _, node := range nodesMap {
		nodes[i] = node
		peers[i] = p2p.NewPeer(node.ID(), node.ID().String(), nil)
		i++
	}
	store := NewPeerStore(newInMemCache(t))
	provider := fakePeerProvider{peers}
	_, err := GetFirstConnected(provider, store)
	require.EqualError(t, ErrNoConnected, err.Error())
	require.NoError(t, store.Update(nodes))
	node, err := GetFirstConnected(provider, store)
	require.NoError(t, err)
	require.Contains(t, nodesMap, node.ID())
}

type trackingNodeNotifee struct {
	calls [][]*enode.Node
}

func (t *trackingNodeNotifee) Notify(nodes []*enode.Node) {
	t.calls = append(t.calls, nodes)
}

func TestEnsureNewRecordsAddedFirst(t *testing.T) {
	notifee := new(trackingNodeNotifee)
	store := NewPeerStore(newInMemCache(t))
	nodes := make([]*enode.Node, 3)
	fillWithRandomNodes(t, nodes)
	require.NoError(t, store.Update(nodes))
	record := NewPeerRecord(nodes[0])
	record.LastUsed = time.Now()
	require.NoError(t, store.cache.UpdateRecord(record))
	require.NoError(t, EnsureUsedRecordsAddedFirst(store, notifee))
	require.Len(t, notifee.calls, 2)
	require.Len(t, notifee.calls[0], 1)
	require.Equal(t, nodes[0].ID(), notifee.calls[0][0].ID())
	require.Len(t, notifee.calls[1], 3)
}
