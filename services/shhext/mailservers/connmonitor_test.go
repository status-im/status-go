package mailservers

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/t/utils"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

func TestUsedConnectionPersisted(t *testing.T) {
	nodes := make([]*enode.Node, 2)
	fillWithRandomNodes(t, nodes)

	cache := newInMemCache(t)
	store := NewPeerStore(cache)
	require.NoError(t, store.Update(nodes))
	whisperMock := newFakeEnvelopesEvents()
	monitor := NewLastUsedConnectionMonitor(store, cache, whisperMock)
	monitor.Start()

	// Send a confirmation that we received history from one of the peers.
	select {
	case whisperMock.input <- whisper.EnvelopeEvent{
		Event: whisper.EventMailServerRequestCompleted, Peer: nodes[0].ID()}:
	case <-time.After(time.Second):
		require.FailNow(t, "can't send a 'completed' event")
	}

	// Wait until records will be updated in the cache.
	require.NoError(t, utils.Eventually(func() error {
		records, err := cache.LoadAll()
		if err != nil {
			return err
		}
		if lth := len(records); lth != 2 {
			return fmt.Errorf("unexpected length of all records stored in the cache. expected %d got %d", 2, lth)
		}
		var used bool
		for _, r := range records {
			if r.Node().ID() == nodes[0].ID() {
				used = !r.LastUsed.IsZero()
			}
		}
		if !used {
			return fmt.Errorf("record %s is not marked as used", nodes[0].ID())
		}
		return nil
	}, time.Second, 100*time.Millisecond))

	// Use different peer, first will be marked as unused.
	select {
	case whisperMock.input <- whisper.EnvelopeEvent{
		Event: whisper.EventMailServerRequestCompleted, Peer: nodes[1].ID()}:
	case <-time.After(time.Second):
		require.FailNow(t, "can't send a 'completed' event")
	}

	require.NoError(t, utils.Eventually(func() error {
		records, err := cache.LoadAll()
		if err != nil {
			return err
		}
		if lth := len(records); lth != 2 {
			return fmt.Errorf("unexpected length of all records stored in the cache. expected %d got %d", 2, lth)
		}
		sort.Slice(records, func(i, j int) bool {
			return records[i].LastUsed.After(records[j].LastUsed)
		})
		if records[0].Node().ID() != nodes[1].ID() {
			return fmt.Errorf("record wasn't updated after previous event")
		}
		return nil
	}, time.Second, 100*time.Millisecond))
}
