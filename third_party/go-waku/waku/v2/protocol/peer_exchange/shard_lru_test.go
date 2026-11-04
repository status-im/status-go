package peer_exchange

import (
	"crypto/ecdsa"
	"testing"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
)

func getEnode(t *testing.T, key *ecdsa.PrivateKey, cluster uint16, indexes ...uint16) *enode.Node {
	if key == nil {
		var err error
		key, err = gcrypto.GenerateKey()
		require.NoError(t, err)
	}
	myNode, err := wenr.NewLocalnode(key)
	require.NoError(t, err)
	if cluster != 0 {
		shard, err := protocol.NewRelayShards(cluster, indexes...)
		require.NoError(t, err)
		myNode.Set(enr.WithEntry(wenr.ShardingBitVectorEnrField, shard.BitVector()))
	}
	return myNode.Node()
}

func TestLruMoreThanSize(t *testing.T) {
	node1 := getEnode(t, nil, 1, 1)
	node2 := getEnode(t, nil, 1, 1, 2)
	node3 := getEnode(t, nil, 1, 1)

	lru := newShardLRU(2)

	err := lru.Add(node1)
	require.NoError(t, err)

	err = lru.Add(node2)
	require.NoError(t, err)

	err = lru.Add(node3)
	require.NoError(t, err)

	nodes := lru.GetRandomNodes(&ShardInfo{1, 1}, 1)

	require.Equal(t, 1, len(nodes))
	if nodes[0].ID() != node2.ID() && nodes[0].ID() != node3.ID() {
		t.Fatalf("different node found %v", nodes)
	}
	// checks if removed node is deleted from lru or not
	require.Nil(t, lru.Get(node1.ID()))

	// node 2 is removed from lru for cluster 1/shard 1 but it is still being maintained for cluster1,index2
	{
		err = lru.Add(getEnode(t, nil, 1, 1)) // add two more nodes to 1/1 cluster shard
		require.NoError(t, err)

		err = lru.Add(getEnode(t, nil, 1, 1))
		require.NoError(t, err)

	}

	// node2 still present in lru
	require.Equal(t, node2, lru.Get(node2.ID()))

	// now node2 is removed from all shards' cache
	{
		err = lru.Add(getEnode(t, nil, 1, 2)) // add two more nodes to 1/2 cluster shard
		require.NoError(t, err)

		err = lru.Add(getEnode(t, nil, 1, 2))
		require.NoError(t, err)
	}

	// node2 still present in lru
	require.Nil(t, lru.Get(node2.ID()))
}

func TestLruNodeWithNewSeq(t *testing.T) {
	lru := newShardLRU(2)
	//
	key, err := gcrypto.GenerateKey()
	require.NoError(t, err)

	node1 := getEnode(t, key, 1, 1)
	err = lru.Add(node1)
	require.NoError(t, err)

	node1 = getEnode(t, key, 1, 2, 3)
	err = lru.Add(node1)
	require.NoError(t, err)

	//
	nodes := lru.GetRandomNodes(&ShardInfo{1, 1}, 2)
	require.Equal(t, 0, len(nodes))
	//
	nodes = lru.GetRandomNodes(&ShardInfo{1, 2}, 2)
	require.Equal(t, 1, len(nodes))
	//
	nodes = lru.GetRandomNodes(&ShardInfo{1, 3}, 2)
	require.Equal(t, 1, len(nodes))
}

func TestLruNoShard(t *testing.T) {
	lru := newShardLRU(2)

	node1 := getEnode(t, nil, 0)
	node2 := getEnode(t, nil, 0)

	err := lru.Add(node1)
	require.NoError(t, err)

	err = lru.Add(node2)
	require.NoError(t, err)

	// check returned nodes
	require.Equal(t, 2, lru.len(nil))
	for _, node := range lru.GetRandomNodes(nil, 2) {
		if node.ID() != node1.ID() && node.ID() != node2.ID() {
			t.Fatalf("different node found %v", node)
		}
	}
}

// checks if lru is able to handle nodes with/without shards together
func TestLruMixedNodes(t *testing.T) {
	lru := newShardLRU(2)

	node1 := getEnode(t, nil, 0)
	err := lru.Add(node1)
	require.NoError(t, err)

	node2 := getEnode(t, nil, 1, 1)
	err = lru.Add(node2)
	require.NoError(t, err)

	node3 := getEnode(t, nil, 1, 2)
	err = lru.Add(node3)
	require.NoError(t, err)

	// check that default
	require.Equal(t, 3, lru.len(nil))

	//
	nodes := lru.GetRandomNodes(&ShardInfo{1, 1}, 2)
	require.Equal(t, 1, len(nodes))
	require.Equal(t, node2.ID(), nodes[0].ID())

	//
	nodes = lru.GetRandomNodes(&ShardInfo{1, 2}, 2)
	require.Equal(t, 1, len(nodes))
	require.Equal(t, node3.ID(), nodes[0].ID())
}
