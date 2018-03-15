package peers

import (
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/geth/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeersRange(t *testing.T) {
	path, err := ioutil.TempDir("/tmp", "status-peers-test-")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(path))
	}()
	rootDB, err := db.CreateDatabase(path)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, rootDB.Close())
	}()

	peersDB := Cache{db: rootDB}
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
