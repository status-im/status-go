package peers

import (
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"

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

	var (
		peers    [3]*discv5.Node
		nodeIDs  [3]enode.ID
		expected = make(map[string]struct{})
	)
	for i := 0; i < len(peers); i++ {
		identity, err := crypto.GenerateKey()
		require.NoError(t, err)
		peers[i] = discv5.NewNode(
			discv5.PubkeyID(&identity.PublicKey),
			net.IPv4(100, 100, 0, byte(i)),
			32311,
			32311)
		nodeIDs[i] = enode.PubkeyToIDV4(&identity.PublicKey)
		expected[peers[i].String()] = struct{}{}
	}
	for _, peer := range peers {
		assert.NoError(t, peersDB.AddPeer(peer, topic))
	}

	nodes := peersDB.GetPeersRange(topic, 3)
	require.Len(t, nodes, 3)

	for _, node := range nodes {
		_, ok := expected[node.String()]
		assert.True(t, ok)
	}

	assert.NoError(t, peersDB.RemovePeer(nodeIDs[0], topic))
	require.Len(t, peersDB.GetPeersRange(topic, 3), 2)
}

func TestMultipleTopics(t *testing.T) {
	peersDB, err := newInMemoryCache()
	require.NoError(t, err)
	topics := []discv5.Topic{discv5.Topic("first"), discv5.Topic("second")}
	for i := range topics {
		for j := 0; j < 3; j++ {
			identity, err := crypto.GenerateKey()
			require.NoError(t, err)
			peer := discv5.NewNode(
				discv5.PubkeyID(&identity.PublicKey),
				net.IPv4(100, 100, 0, byte(i)),
				32311,
				32311)
			assert.NoError(t, peersDB.AddPeer(peer, topics[i]))
		}
	}
	for i := range topics {
		nodes := peersDB.GetPeersRange(topics[i], 10)
		assert.Len(t, nodes, 3)
	}
}
