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

func TestPeersRange(t *testing.T) {
	peersDB, err := newInMemoryCache()
	require.NoError(t, err)
	topic := discv5.Topic("test")

	peerCount := 3
	peers, err := createDiscv5Peers(peerCount)
	require.NoError(t, err)

	addedPeerIDs := make(map[string]struct{})
	for _, peer := range peers {
		assert.NoError(t, peersDB.AddPeer(peer, topic))
		addedPeerIDs[peer.String()] = struct{}{}
	}

	// check if all peers are cached
	cachedNodes := peersDB.GetPeersRange(topic, peerCount)
	require.Len(t, cachedNodes, peerCount)
	// and returned peers are these we just created
	for _, node := range cachedNodes {
		_, ok := addedPeerIDs[node.String()]
		assert.True(t, ok)
	}

	// test removing peers
	pk, err := peers[0].ID.Pubkey()
	require.NoError(t, err)
	require.NoError(t, peersDB.RemovePeer(enode.PubkeyToIDV4(pk), topic))
	cachedNodes = peersDB.GetPeersRange(topic, peerCount)
	require.Len(t, cachedNodes, peerCount-1)
}

func TestMultipleTopics(t *testing.T) {
	peersDB, err := newInMemoryCache()
	require.NoError(t, err)
	topics := []discv5.Topic{discv5.Topic("first"), discv5.Topic("second")}

	peerCount := 3
	peers, err := createDiscv5Peers(peerCount)
	require.NoError(t, err)

	for i := range topics {
		for _, peer := range peers {
			assert.NoError(t, peersDB.AddPeer(peer, topics[i]))
		}
	}

	for i := range topics {
		nodes := peersDB.GetPeersRange(topics[i], peerCount+1)
		assert.Len(t, nodes, peerCount)
	}
}

// newInMemoryCache creates a cache for tests
func newInMemoryCache() (*Cache, error) {
	memdb, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	return NewCache(memdb), nil
}

func createDiscv5Peers(count int) ([]*discv5.Node, error) {
	nodes := make([]*discv5.Node, count)

	for i := 0; i < count; i++ {
		id, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}

		nodes[i] = discv5.NewNode(
			discv5.PubkeyID(&id.PublicKey),
			net.IPv4(10, 10, 0, byte(i)),
			32311,
			32311,
		)
	}

	return nodes, nil
}
