package mailservers

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/t/utils"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/assert"
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
	input chan whisper.EnvelopeEvent
}

func (f *fakeEnvelopeEvents) SubscribeEnvelopeEvents(output chan<- whisper.EnvelopeEvent) event.Subscription {
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

func newFakeEnvelopesEvents() *fakeEnvelopeEvents {
	return &fakeEnvelopeEvents{
		input: make(chan whisper.EnvelopeEvent),
	}
}

func fillWithRandomNodes(t *testing.T, nodes []*enode.Node) {
	var err error
	for i := range nodes {
		nodes[i], err = RandomNode()
		require.NoError(t, err)
	}
}

func getMapWithRandomNodes(t *testing.T, n int) map[enode.ID]*enode.Node {
	nodes := make([]*enode.Node, n)
	fillWithRandomNodes(t, nodes)
	return nodesToMap(nodes)
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
			getMapWithRandomNodes(t, 0),
			getMapWithRandomNodes(t, 3),
			2,
		},
		{
			"FullReplace",
			getMapWithRandomNodes(t, 3),
			getMapWithRandomNodes(t, 3),
			2,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			peers := newFakePeerAdderRemover()
			state := newInternalState(peers, tc.target, 0)
			state.replaceNodes(tc.old)
			require.Len(t, peers.nodes, len(tc.old))
			for n := range peers.nodes {
				require.Contains(t, tc.old, n)
			}
			state.replaceNodes(tc.new)
			require.Len(t, peers.nodes, len(tc.new))
			for n := range peers.nodes {
				require.Contains(t, tc.new, n)
			}
		})
	}
}

func TestPartialReplaceNodesBelowTarget(t *testing.T) {
	peers := newFakePeerAdderRemover()
	old := getMapWithRandomNodes(t, 1)
	new := getMapWithRandomNodes(t, 2)
	state := newInternalState(peers, 2, 0)
	state.replaceNodes(old)
	mergeOldIntoNew(old, new)
	state.replaceNodes(new)
	require.Len(t, peers.nodes, len(new))
}

func TestPartialReplaceNodesAboveTarget(t *testing.T) {
	peers := newFakePeerAdderRemover()
	initial, err := RandomNode()
	require.NoError(t, err)
	old := nodesToMap([]*enode.Node{initial})
	new := getMapWithRandomNodes(t, 2)
	state := newInternalState(peers, 1, 0)
	state.replaceNodes(old)
	state.nodeAdded(initial.ID())
	mergeOldIntoNew(old, new)
	state.replaceNodes(new)
	require.Len(t, peers.nodes, 1)
}

func TestConnectionManagerAddDrop(t *testing.T) {
	server := newFakeServer()
	whisper := newFakeEnvelopesEvents()
	target := 1
	connmanager := NewConnectionManager(server, whisper, target, 1, 0)
	connmanager.Start()
	defer connmanager.Stop()
	nodes := []*enode.Node{}
	for _, n := range getMapWithRandomNodes(t, 3) {
		nodes = append(nodes, n)
	}
	// Send 3 random nodes to connection manager.
	connmanager.Notify(nodes)
	var initial enode.ID
	// Wait till connection manager establishes connection with 1 peer.
	require.NoError(t, utils.Eventually(func() error {
		nodes := server.Nodes()
		if len(nodes) != target {
			return fmt.Errorf("unexpected number of connected servers: %d", len(nodes))
		}
		initial = nodes[0]
		return nil
	}, time.Second, 100*time.Millisecond))
	// Send an event that peer was dropped.
	select {
	case server.input <- &p2p.PeerEvent{Peer: initial, Type: p2p.PeerEventTypeDrop}:
	case <-time.After(time.Second):
		require.FailNow(t, "can't send a drop event")
	}
	// Connection manager should establish connection with any other peer from initial list.
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
	connmanager := NewConnectionManager(server, whisper, target, 1, 0)
	connmanager.Start()
	defer connmanager.Stop()
	nodes := []*enode.Node{}
	for _, n := range getMapWithRandomNodes(t, 3) {
		nodes = append(nodes, n)
	}
	// Send a single node to connection manager.
	connmanager.Notify(nodes[:1])
	// Wait until this node will get connected.
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
	// Replace previously sent node with 2 different nodes.
	connmanager.Notify(nodes[1:])
	// Wait until connection manager replaces node connected in the first round.
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

func setupTestConnectionAfterExpiry(t *testing.T, server *fakePeerEvents, whisperMock *fakeEnvelopeEvents, target, maxFailures int, hash common.Hash) (*ConnectionManager, enode.ID) {
	connmanager := NewConnectionManager(server, whisperMock, target, maxFailures, 0)
	connmanager.Start()
	nodes := []*enode.Node{}
	for _, n := range getMapWithRandomNodes(t, 2) {
		nodes = append(nodes, n)
	}
	// Send two random nodes to connection manager.
	connmanager.Notify(nodes)
	var initial enode.ID
	// Wait until connection manager establishes connection with one node.
	require.NoError(t, utils.Eventually(func() error {
		nodes := server.Nodes()
		if len(nodes) != target {
			return fmt.Errorf("unexpected number of connected servers: %d", len(nodes))
		}
		initial = nodes[0]
		return nil
	}, time.Second, 100*time.Millisecond))
	// Send event that history request for connected peer was sent.
	select {
	case whisperMock.input <- whisper.EnvelopeEvent{
		Event: whisper.EventMailServerRequestSent, Peer: initial, Hash: hash}:
	case <-time.After(time.Second):
		require.FailNow(t, "can't send a 'sent' event")
	}
	return connmanager, initial
}

func TestConnectionChangedAfterExpiry(t *testing.T) {
	server := newFakeServer()
	whisperMock := newFakeEnvelopesEvents()
	target := 1
	maxFailures := 1
	hash := common.Hash{1}
	connmanager, initial := setupTestConnectionAfterExpiry(t, server, whisperMock, target, maxFailures, hash)
	defer connmanager.Stop()

	// And eventually expired.
	select {
	case whisperMock.input <- whisper.EnvelopeEvent{
		Event: whisper.EventMailServerRequestExpired, Peer: initial, Hash: hash}:
	case <-time.After(time.Second):
		require.FailNow(t, "can't send an 'expiry' event")
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

func TestConnectionChangedAfterSecondExpiry(t *testing.T) {
	server := newFakeServer()
	whisperMock := newFakeEnvelopesEvents()
	target := 1
	maxFailures := 2
	hash := common.Hash{1}
	connmanager, initial := setupTestConnectionAfterExpiry(t, server, whisperMock, target, maxFailures, hash)
	defer connmanager.Stop()

	// First expired is sent. Nothing should happen.
	select {
	case whisperMock.input <- whisper.EnvelopeEvent{
		Event: whisper.EventMailServerRequestExpired, Peer: initial, Hash: hash}:
	case <-time.After(time.Second):
		require.FailNow(t, "can't send an 'expiry' event")
	}

	// we use 'eventually' as 'consistently' because this function will retry for a given timeout while error is received
	require.EqualError(t, utils.Eventually(func() error {
		nodes := server.Nodes()
		if len(nodes) != target {
			return fmt.Errorf("unexpected number of connected servers: %d", len(nodes))
		}
		if nodes[0] == initial {
			return fmt.Errorf("connected node wasn't changed from %s", initial)
		}
		return nil
	}, time.Second, 100*time.Millisecond), fmt.Sprintf("connected node wasn't changed from %s", initial))

	// second expiry event
	select {
	case whisperMock.input <- whisper.EnvelopeEvent{
		Event: whisper.EventMailServerRequestExpired, Peer: initial, Hash: hash}:
	case <-time.After(time.Second):
		require.FailNow(t, "can't send an 'expiry' event")
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

func TestProcessReplacementWaitsForConnections(t *testing.T) {
	srv := newFakePeerAdderRemover()
	target := 1
	timeout := time.Second
	nodes := make([]*enode.Node, 2)
	fillWithRandomNodes(t, nodes)
	events := make(chan *p2p.PeerEvent)
	state := newInternalState(srv, target, timeout)
	state.currentNodes = nodesToMap(nodes)
	go func() {
		select {
		case events <- &p2p.PeerEvent{Peer: nodes[0].ID(), Type: p2p.PeerEventTypeAdd}:
		case <-time.After(time.Second):
			assert.FailNow(t, "can't send a drop event")
		}
	}()
	state.processReplacement(nodes, events)
	require.Len(t, state.connected, 1)
}
