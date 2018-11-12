package peers

import (
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
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
	ids := generateCorrectDiscv5Nodeid(3)
	t.Log("ids")
	for i := range ids {
		t.Log(ids[i])
	}
	t.Log("ids")
	peers := [3]*discv5.Node{
		discv5.NewNode(ids[1], net.IPv4(100, 100, 0, 3), 32311, 32311),
		discv5.NewNode(ids[2], net.IPv4(100, 100, 0, 4), 32311, 32311),
		discv5.NewNode(ids[0], net.IPv4(100, 100, 0, 2), 32311, 32311),
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

	id, err := Discv5IDToEnodeID(peers[1].ID)
	require.NoError(t, err)
	assert.NoError(t, peersDB.RemovePeer(id, topic))
	require.Len(t, peersDB.GetPeersRange(topic, 3), 2)
}

func TestMultipleTopics(t *testing.T) {
	peersDB, err := newInMemoryCache()
	require.NoError(t, err)
	topics := []discv5.Topic{discv5.Topic("first"), discv5.Topic("second")}
	keys := make(map[discv5.Topic]map[discv5.NodeID]struct{})

	for i := range topics {
		var peers []*discv5.Node
		keys[topics[i]] = make(map[discv5.NodeID]struct{})

		for j := 0; j < 3; j++ {
			key, _ := crypto.GenerateKey()
			nodeID := discv5.PubkeyID(&key.PublicKey)
			keys[topics[i]][nodeID] = struct{}{}
			node := discv5.NewNode(nodeID, net.IPv4(100, 100, 0, byte(j+1)), 32311, 32311)
			peers = append(peers, node)
		}

		for _, peer := range peers {
			assert.NoError(t, peersDB.AddPeer(peer, topics[i]))
		}
	}

	for i := range topics {
		nodes := peersDB.GetPeersRange(topics[i], 10)
		assert.Len(t, nodes, 3)
		for _, n := range nodes {
			_, ok := keys[topics[i]][n.ID]
			assert.True(t, ok)
		}
	}
}
