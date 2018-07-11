package peers

import (
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

// newInMemoryCache creates a cache for tests
func newInMemoryCache() (*Cache, error) {
	memdb, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return NewCache(memdb), nil
}

func TestPeersRange(t *testing.T) {
	peersDB, err := newInMemoryCache()
	require.NoError(t, err)
	topic := discv5.Topic("test")
	peers := [3]*discv5.Node{
		discv5.NewNode(discv5.NodeID{3}, net.IPv4(100, 100, 0, 3), 32311, 32311),
		discv5.NewNode(discv5.NodeID{4}, net.IPv4(100, 100, 0, 4), 32311, 32311),
		discv5.NewNode(discv5.NodeID{2}, net.IPv4(100, 100, 0, 2), 32311, 32311),
	}
	for _, peer := range peers {
		assert.NoError(t, peersDB.AddPeer(peer, topic))
	}
	nodes := peersDB.GetPeersRange(topic, 3)
	require.Len(t, nodes, 3)
	// object will be ordered by memcpy order of bytes 2,3,4 in our case
	// order of tests is intentionally mixed to make it obvious that range is
	// not ordered by the insertion time
	assert.Equal(t, peers[2].String(), nodes[0].String())
	assert.Equal(t, peers[0].String(), nodes[1].String())
	assert.Equal(t, peers[1].String(), nodes[2].String())

	assert.NoError(t, peersDB.RemovePeer(peers[1].ID, topic))
	require.Len(t, peersDB.GetPeersRange(topic, 3), 2)
}

func TestMultipleTopics(t *testing.T) {
	peersDB, err := newInMemoryCache()
	require.NoError(t, err)
	topics := []discv5.Topic{discv5.Topic("first"), discv5.Topic("second")}
	for i := range topics {
		peers := [3]*discv5.Node{
			discv5.NewNode(discv5.NodeID{byte(i), 1}, net.IPv4(100, 100, 0, 3), 32311, 32311),
			discv5.NewNode(discv5.NodeID{byte(i), 2}, net.IPv4(100, 100, 0, 4), 32311, 32311),
			discv5.NewNode(discv5.NodeID{byte(i), 3}, net.IPv4(100, 100, 0, 2), 32311, 32311)}
		for _, peer := range peers {
			assert.NoError(t, peersDB.AddPeer(peer, topics[i]))
		}
	}
	for i := range topics {
		nodes := peersDB.GetPeersRange(topics[i], 10)
		assert.Len(t, nodes, 3)
		for _, n := range nodes {
			assert.Equal(t, byte(i), n.ID[0])
		}
	}
}
