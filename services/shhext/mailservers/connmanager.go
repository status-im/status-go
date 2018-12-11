package mailservers

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	whisper "github.com/status-im/whisper/whisperv6"
)

const (
	peerEventsBuffer    = 10 // sufficient buffer to avoid blocking a p2p feed.
	whisperEventsBuffer = 20 // sufficient buffer to avod blocking a whisper envelopes feed.
)

// PeerAdderRemover is an interface for adding or removing peers.
type PeerAdderRemover interface {
	AddPeer(node *enode.Node)
	RemovePeer(node *enode.Node)
}

// PeerEventsSubscriber interface to subscribe for p2p.PeerEvent's.
type PeerEventsSubscriber interface {
	SubscribeEvents(chan *p2p.PeerEvent) event.Subscription
}

// EnvelopeEventSubscbriber interface to subscribe for whisper.EnvelopeEvent's.
type EnvelopeEventSubscbriber interface {
	SubscribeEnvelopeEvents(chan<- whisper.EnvelopeEvent) event.Subscription
}

type p2pServer interface {
	PeerAdderRemover
	PeerEventsSubscriber
}

// NewConnectionManager creates an instance of ConnectionManager.
func NewConnectionManager(server p2pServer, whisper EnvelopeEventSubscbriber, target int, timeout time.Duration) *ConnectionManager {
	return &ConnectionManager{
		server:           server,
		whisper:          whisper,
		connectedTarget:  target,
		notifications:    make(chan []*enode.Node),
		timeoutWaitAdded: timeout,
	}
}

// ConnectionManager manages keeps target of peers connected.
type ConnectionManager struct {
	wg   sync.WaitGroup
	quit chan struct{}

	server  p2pServer
	whisper EnvelopeEventSubscbriber

	notifications    chan []*enode.Node
	connectedTarget  int
	timeoutWaitAdded time.Duration
}

// Notify sends a non-blocking notification about new nodes.
func (ps *ConnectionManager) Notify(nodes []*enode.Node) {
	ps.wg.Add(1)
	go func() {
		select {
		case ps.notifications <- nodes:
		case <-ps.quit:
		}
		ps.wg.Done()
	}()

}

// Start subscribes to a p2p server and handles new peers and state updates for those peers.
func (ps *ConnectionManager) Start() {
	ps.quit = make(chan struct{})
	ps.wg.Add(1)
	go func() {
		state := newInternalState(ps.server, ps.connectedTarget, ps.timeoutWaitAdded)

		events := make(chan *p2p.PeerEvent, peerEventsBuffer)
		sub := ps.server.SubscribeEvents(events)
		whisperEvents := make(chan whisper.EnvelopeEvent, whisperEventsBuffer)
		whisperSub := ps.whisper.SubscribeEnvelopeEvents(whisperEvents)
		requests := map[common.Hash]struct{}{}

		defer sub.Unsubscribe()
		defer whisperSub.Unsubscribe()
		defer ps.wg.Done()
		for {
			select {
			case <-ps.quit:
				return
			case err := <-sub.Err():
				log.Error("retry after error subscribing to p2p events", "error", err)
				return
			case err := <-whisperSub.Err():
				log.Error("retry after error suscribing to whisper events", "error", err)
				return
			case newNodes := <-ps.notifications:
				state.processReplacement(newNodes, events)
			case ev := <-events:
				processPeerEvent(state, ev)
			case ev := <-whisperEvents:
				// TODO what about completed but with error? what about expired envelopes?
				switch ev.Event {
				case whisper.EventMailServerRequestSent:
					requests[ev.Hash] = struct{}{}
				case whisper.EventMailServerRequestCompleted:
					delete(requests, ev.Hash)
				case whisper.EventMailServerRequestExpired:
					_, exist := requests[ev.Hash]
					if !exist {
						continue
					}
					log.Debug("request to a mail server expired, disconncet a peer", "address", ev.Peer)
					state.nodeDisconnected(ev.Peer)
				}
			}
		}
	}()
}

// Stop gracefully closes all background goroutines and waits until they finish.
func (ps *ConnectionManager) Stop() {
	if ps.quit == nil {
		return
	}
	select {
	case <-ps.quit:
		return
	default:
	}
	close(ps.quit)
	ps.wg.Wait()
	ps.quit = nil
}

func (state *internalState) processReplacement(newNodes []*enode.Node, events <-chan *p2p.PeerEvent) {
	replacement := map[enode.ID]*enode.Node{}
	for _, n := range newNodes {
		replacement[n.ID()] = n
	}
	state.replaceNodes(replacement)
	if state.ReachedTarget() {
		log.Debug("already connected with required target", "target", state.target)
		return
	}
	if state.timeout != 0 {
		log.Debug("waiting defined timeout to establish connections",
			"timeout", state.timeout, "target", state.target)
		timer := time.NewTimer(state.timeout)
		waitForConnections(state, timer.C, events)
		timer.Stop()
	}
}

func newInternalState(srv PeerAdderRemover, target int, timeout time.Duration) *internalState {
	return &internalState{
		options:      options{target: target, timeout: timeout},
		srv:          srv,
		connected:    map[enode.ID]struct{}{},
		currentNodes: map[enode.ID]*enode.Node{},
	}
}

type options struct {
	target  int
	timeout time.Duration
}

type internalState struct {
	options
	srv PeerAdderRemover

	connected    map[enode.ID]struct{}
	currentNodes map[enode.ID]*enode.Node
}

func (state *internalState) ReachedTarget() bool {
	return len(state.connected) >= state.target
}

func (state *internalState) replaceNodes(new map[enode.ID]*enode.Node) {
	for nid, n := range state.currentNodes {
		if _, exist := new[nid]; !exist {
			if _, exist := state.connected[nid]; exist {
				delete(state.connected, nid)
			}
			state.srv.RemovePeer(n)
		}
	}
	if !state.ReachedTarget() {
		for _, n := range new {
			state.srv.AddPeer(n)
		}
	}
	state.currentNodes = new
}

func (state *internalState) nodeAdded(peer enode.ID) {
	n, exist := state.currentNodes[peer]
	if !exist {
		return
	}
	if state.ReachedTarget() {
		state.srv.RemovePeer(n)
	} else {
		state.connected[n.ID()] = struct{}{}
	}
}

func (state *internalState) nodeDisconnected(peer enode.ID) {
	n, exist := state.currentNodes[peer] // unrelated event
	if !exist {
		return
	}
	_, exist = state.connected[peer] // check if already disconnected
	if !exist {
		return
	}
	if len(state.currentNodes) == 1 { // keep node connected if we don't have another choice
		return
	}
	state.srv.RemovePeer(n) // remove peer permanently, otherwise p2p.Server will try to reconnect
	delete(state.connected, peer)
	if !state.ReachedTarget() { // try to connect with any other selected (but not connected) node
		for nid, n := range state.currentNodes {
			_, exist := state.connected[nid]
			if exist || peer == nid {
				continue
			}
			state.srv.AddPeer(n)
		}
	}
}

func processPeerEvent(state *internalState, ev *p2p.PeerEvent) {
	switch ev.Type {
	case p2p.PeerEventTypeAdd:
		log.Debug("connected to a mailserver", "address", ev.Peer)
		state.nodeAdded(ev.Peer)
	case p2p.PeerEventTypeDrop:
		log.Debug("mailserver disconnected", "address", ev.Peer)
		state.nodeDisconnected(ev.Peer)
	}
}

func waitForConnections(state *internalState, timeout <-chan time.Time, events <-chan *p2p.PeerEvent) {
	for {
		select {
		case ev := <-events:
			processPeerEvent(state, ev)
			if state.ReachedTarget() {
				return
			}
		case <-timeout:
			return
		}
	}

}
