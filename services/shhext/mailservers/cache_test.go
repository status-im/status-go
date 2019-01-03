package mailservers

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

func newInMemCache(t *testing.T) *Cache {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)
	return NewCache(db)
}

func containsNode(nodes []*enode.Node, node *enode.Node) error {
	for _, n := range nodes {
		if n.ID() == node.ID() {
			return nil
		}
	}
	return fmt.Errorf("nodes %+s do not contain  %s", nodes, node)
}

func TestReplaceRecords(t *testing.T) {
	nodesNumber := 3
	cache := newInMemCache(t)
	nodes := make([]*enode.Node, nodesNumber)
	// First round is a sanity check that records were written.
	fillWithRandomNodes(t, nodes)
	require.NoError(t, cache.Replace(nodes))
	records, err := cache.LoadAll()
	require.NoError(t, err)
	require.Len(t, records, nodesNumber)
	for i := range records {
		require.NoError(t, containsNode(nodes, records[i].Node()))
	}
	// Replace all nodes and verify that length is the same and loaded records are found.
	fillWithRandomNodes(t, nodes)
	require.NoError(t, cache.Replace(nodes))
	records, err = cache.LoadAll()
	require.NoError(t, err)
	require.Len(t, records, nodesNumber)
	for i := range records {
		require.NoError(t, containsNode(nodes, records[i].Node()))
	}
}

func TestUsedRecord(t *testing.T) {
	cache := newInMemCache(t)
	node, err := RandomNode()
	require.NoError(t, err)
	record := PeerRecord{node: node}
	require.NoError(t, cache.UpdateRecord(record))
	record.LastUsed = time.Now()
	require.NoError(t, cache.UpdateRecord(record))
	records, err := cache.LoadAll()
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.True(t, record.LastUsed.Equal(records[0].LastUsed))
}

func TestTimestampPreservedOnReplace(t *testing.T) {
	cache := newInMemCache(t)
	node, err := RandomNode()
	require.NoError(t, err)
	record := PeerRecord{node: node, LastUsed: time.Now()}
	require.NoError(t, cache.UpdateRecord(record))
	require.NoError(t, cache.Replace([]*enode.Node{node}))
	records, err := cache.LoadAll()
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, node.ID(), records[0].Node().ID())
	require.False(t, records[0].LastUsed.IsZero(), "timestamp should be preserved and not equal to zero")

}
