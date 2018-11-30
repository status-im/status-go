package mailservers

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

type fakePeerEvents struct {
	mu    sync.Mutex
	nodes map[enode.ID]struct{}
	input chan *p2p.PeerEvent
}

func (f *fakePeerEvents) Nodes() []enode.ID {
	f.mu.Lock()
	rst := make([]enode.ID, 0, len(f.nodes))
	for n := range f.nodes {
		rst = append(rst, n)
	}
	f.mu.Unlock()
	return rst
}

func (f *fakePeerEvents) AddPeer(node *enode.Node) {
	f.mu.Lock()
	f.nodes[node.ID()] = struct{}{}
	f.mu.Unlock()
	if f.input == nil {
		return
	}
	f.input <- &p2p.PeerEvent{
		Peer: node.ID(),
		Type: p2p.PeerEventTypeAdd,
	}
}

func (f *fakePeerEvents) RemovePeer(node *enode.Node) {
	f.mu.Lock()
	delete(f.nodes, node.ID())
	f.mu.Unlock()
	if f.input == nil {
		return
	}
	f.input <- &p2p.PeerEvent{
		Peer: node.ID(),
		Type: p2p.PeerEventTypeDrop,
	}
}

func newFakePeerAdderRemover() *fakePeerEvents {
	return &fakePeerEvents{nodes: map[enode.ID]struct{}{}}
}

func (f *fakePeerEvents) SubscribeEvents(output chan *p2p.PeerEvent) event.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		for {
			select {
			case <-quit:
				return nil
			case ev := <-f.input:
				// will block the same way as in any feed
				output <- ev
			}
		}
	})
}

func newFakeServer() *fakePeerEvents {
	srv := newFakePeerAdderRemover()
	srv.input = make(chan *p2p.PeerEvent, 20)
	return srv
}

type fakeEnvelopeEvents struct {
	input chan whisperv6.EnvelopeEvent
}

func (f fakeEnvelopeEvents) SubscribeEnvelopeEvents(output chan<- whisperv6.EnvelopeEvent) event.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		for {
			select {
			case <-quit:
				return nil
			case ev := <-f.input:
				// will block the same way as in any feed
				output <- ev
			}
		}
	})
}

func newFakeEnvelopesEvents() fakeEnvelopeEvents {
	return fakeEnvelopeEvents{
		input: make(chan whisperv6.EnvelopeEvent),
	}
}

func getNRandomNodes(t *testing.T, n int) map[enode.ID]*enode.Node {
	rst := map[enode.ID]*enode.Node{}
	for i := 0; i < n; i++ {
		n, err := RandomeNode()
		require.NoError(t, err)
		rst[n.ID()] = n
	}
	return rst
}

func mergeOldIntoNew(old, new map[enode.ID]*enode.Node) {
	for n := range old {
		new[n] = old[n]
	}
}

func TestReplaceNodes(t *testing.T) {
	type testCase struct {
		description string
		old         map[enode.ID]*enode.Node
		new         map[enode.ID]*enode.Node
		target      int
	}
	for _, tc := range []testCase{
		{
			"InitialReplace",
			getNRandomNodes(t, 0),
			getNRandomNodes(t, 3),
			2,
		},
		{
			"FullReplace",
			getNRandomNodes(t, 3),
			getNRandomNodes(t, 3),
			2,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			peers := newFakePeerAdderRemover()
			replaceNodes(peers, tc.target, peers.nodes, nil, tc.old)
			require.Len(t, peers.nodes, len(tc.old))
			for n := range peers.nodes {
				require.Contains(t, tc.old, n)
			}
			replaceNodes(peers, tc.target, peers.nodes, tc.old, tc.new)
			require.Len(t, peers.nodes, len(tc.new))
			for n := range peers.nodes {
				require.Contains(t, tc.new, n)
			}
		})
	}
}

func TestPartialReplaceNodesBelowTarget(t *testing.T) {
	peers := newFakePeerAdderRemover()
	old := getNRandomNodes(t, 1)
	new := getNRandomNodes(t, 2)
	replaceNodes(peers, 2, peers.nodes, nil, old)
	mergeOldIntoNew(old, new)
	replaceNodes(peers, 2, peers.nodes, old, new)
	require.Len(t, peers.nodes, len(new))
}

func TestPartialReplaceNodesAboveTarget(t *testing.T) {
	peers := newFakePeerAdderRemover()
	old := getNRandomNodes(t, 1)
	new := getNRandomNodes(t, 2)
	replaceNodes(peers, 1, peers.nodes, nil, old)
	mergeOldIntoNew(old, new)
	replaceNodes(peers, 1, peers.nodes, old, new)
	require.Len(t, peers.nodes, 1)
}

func TestConnectionManagerAddDrop(t *testing.T) {
	server := newFakeServer()
	whisper := newFakeEnvelopesEvents()
	target := 1
	connmanager := NewConnectionManager(server, whisper, target)
	connmanager.Start()
	defer connmanager.Stop()
	nodes := []*enode.Node{}
	for _, n := range getNRandomNodes(t, 3) {
		nodes = append(nodes, n)
	}
	connmanager.Notify(nodes)
	var initial enode.ID
	require.NoError(t, utils.Eventually(func() error {
		nodes := server.Nodes()
		if len(nodes) != target {
			return fmt.Errorf("unexpected number of connected servers: %d", len(nodes))
		}
		initial = nodes[0]
		return nil
	}, time.Second, 100*time.Millisecond))
	select {
	case server.input <- &p2p.PeerEvent{Peer: initial, Type: p2p.PeerEventTypeDrop}:
	case <-time.After(time.Second):
		require.FailNow(t, "can't send a drop event")
	}
	require.NoError(t, utils.Eventually(func() error {
		nodes := server.Nodes()
		if len(nodes) != target {
			return fmt.Errorf("unexpected number of connected servers: %d", len(nodes))
		}
		if nodes[0] == initial {
			return fmt.Errorf("connected node wasn't changed from %s", initial)
		}
		return nil
	}, time.Second, 100*time.Millisecond))
}

func TestConnectionManagerReplace(t *testing.T) {
	server := newFakeServer()
	whisper := newFakeEnvelopesEvents()
	target := 1
	connmanager := NewConnectionManager(server, whisper, target)
	connmanager.Start()
	defer connmanager.Stop()
	nodes := []*enode.Node{}
	for _, n := range getNRandomNodes(t, 3) {
		nodes = append(nodes, n)
	}
	connmanager.Notify(nodes[:1])
	require.NoError(t, utils.Eventually(func() error {
		connected := server.Nodes()
		if len(connected) != target {
			return fmt.Errorf("unexpected number of connected servers: %d", len(connected))
		}
		if nodes[0].ID() != connected[0] {
			return fmt.Errorf("connected with a wrong peer. expected %s, got %s", nodes[0].ID(), connected[0])
		}
		return nil
	}, time.Second, 100*time.Millisecond))
	connmanager.Notify(nodes[1:])
	require.NoError(t, utils.Eventually(func() error {
		connected := server.Nodes()
		if len(connected) != target {
			return fmt.Errorf("unexpected number of connected servers: %d", len(connected))
		}
		switch connected[0] {
		case nodes[1].ID():
		case nodes[2].ID():
		default:
			return fmt.Errorf("connected with unexpected peer. got %s, expected %+v", connected[0], nodes[1:])
		}
		return nil
	}, time.Second, 100*time.Millisecond))
}

func TestConnectionChangedAfterExpiry(t *testing.T) {
	server := newFakeServer()
	whisper := newFakeEnvelopesEvents()
	target := 1
	connmanager := NewConnectionManager(server, whisper, target)
	connmanager.Start()
	defer connmanager.Stop()
	nodes := []*enode.Node{}
	for _, n := range getNRandomNodes(t, 2) {
		nodes = append(nodes, n)
	}
	connmanager.Notify(nodes)
	var initial enode.ID
	require.NoError(t, utils.Eventually(func() error {
		nodes := server.Nodes()
		if len(nodes) != target {
			return fmt.Errorf("unexpected number of connected servers: %d", len(nodes))
		}
		initial = nodes[0]
		return nil
	}, time.Second, 100*time.Millisecond))
	select {
	case whisper.input <- whisperv6.EnvelopeEvent{Event: whisperv6.EventMailServerRequestExpired, Peer: initial}:
	case <-time.After(time.Second):
		require.FailNow(t, "can't send an expiry event")
	}
	require.NoError(t, utils.Eventually(func() error {
		nodes := server.Nodes()
		if len(nodes) != target {
			return fmt.Errorf("unexpected number of connected servers: %d", len(nodes))
		}
		if nodes[0] == initial {
			return fmt.Errorf("connected node wasn't changed from %s", initial)
		}
		return nil
	}, time.Second, 100*time.Millisecond))
}
