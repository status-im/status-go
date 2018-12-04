package mailservers

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/whisper/whisperv6"
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
	SubscribeEnvelopeEvents(chan<- whisperv6.EnvelopeEvent) event.Subscription
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
		current := map[enode.ID]*enode.Node{}
		connected := map[enode.ID]struct{}{}
		events := make(chan *p2p.PeerEvent, peerEventsBuffer)
		sub := ps.server.SubscribeEvents(events)
		whisperEvents := make(chan whisperv6.EnvelopeEvent, whisperEventsBuffer)
		whisperSub := ps.whisper.SubscribeEnvelopeEvents(whisperEvents)
		requests := map[common.Hash]struct{}{}
		for {
			select {
			case <-ps.quit:
				sub.Unsubscribe()
				whisperSub.Unsubscribe()
				ps.wg.Done()
				return
			case err := <-sub.Err():
				log.Error("retry after error subscribing to p2p events", "error", err)
				sub = ps.server.SubscribeEvents(events)
			case err := <-whisperSub.Err():
				log.Error("retry after error suscribing to whisper events", "error", err)
				whisperSub = ps.whisper.SubscribeEnvelopeEvents(whisperEvents)
			case newNodes := <-ps.notifications:
				current = processReplacement(ps.server, ps.connectedTarget, ps.timeoutWaitAdded, newNodes, events, connected, current)
			case ev := <-events:
				processPeerEvent(ev, ps.server, ps.connectedTarget, connected, current)
			case ev := <-whisperEvents:
				// TODO what about completed but with error? what about expired envelopes?
				switch ev.Event {
				case whisperv6.EventMailServerRequestSent:
					requests[ev.Hash] = struct{}{}
				case whisperv6.EventMailServerRequestCompleted:
					delete(requests, ev.Hash)
				case whisperv6.EventMailServerRequestExpired:
					_, exist := requests[ev.Hash]
					if !exist {
						continue
					}
					log.Debug("request to a mail server expired, disconncet a peer", "address", ev.Peer)
					nodeDisconnected(ps.server, ev.Peer, ps.connectedTarget, connected, current)
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

func processReplacement(srv PeerAdderRemover, target int, timeout time.Duration, newNodes []*enode.Node, events <-chan *p2p.PeerEvent, connected map[enode.ID]struct{}, current map[enode.ID]*enode.Node) map[enode.ID]*enode.Node {
	replacement := map[enode.ID]*enode.Node{}
	for _, n := range newNodes {
		replacement[n.ID()] = n
	}
	replaceNodes(srv, target, connected, current, replacement)
	if target == len(connected) {
		log.Debug("already connected with required target", "target", target)
		return replacement
	}
	if timeout != 0 {
		log.Debug("waiting defined timeout to establish connections",
			"timeout", timeout, "target", target)
		timer := time.NewTimer(timeout)
		waitForConnections(timeout.C, events, srv, target, connected, current)
		timer.Stop()
	}
	return replacement
}

func replaceNodes(srv PeerAdderRemover, target int, connected map[enode.ID]struct{}, old, new map[enode.ID]*enode.Node) {
	for nid, n := range old {
		if _, exist := new[nid]; !exist {
			if _, exist := connected[nid]; exist {
				delete(connected, nid)
			}
			srv.RemovePeer(n)
		}
	}
	if len(connected) < target {
		for _, n := range new {
			srv.AddPeer(n)
		}
	}
}

func nodeAdded(srv PeerAdderRemover, peer enode.ID, target int, connected map[enode.ID]struct{}, nodes map[enode.ID]*enode.Node) {
	n, exist := nodes[peer]
	if !exist {
		return
	}
	if len(connected) == target {
		srv.RemovePeer(n)
	} else {
		connected[n.ID()] = struct{}{}
	}
}

func nodeDisconnected(srv PeerAdderRemover, peer enode.ID, target int, connected map[enode.ID]struct{}, nodes map[enode.ID]*enode.Node) {
	n, exist := nodes[peer] // unrelated event
	if !exist {
		return
	}
	_, exist = connected[peer] // check if already disconnected
	if !exist {
		return
	}
	if len(nodes) == 1 { // keep node connected if we don't have another choice
		return
	}
	srv.RemovePeer(n) // remove peer permanently, otherwise p2p.Server will try to reconnect
	delete(connected, peer)
	if len(connected) < target { // try to connect with any other selected (but not connected) node
		for nid, n := range nodes {
			_, exist := connected[nid]
			if exist || peer == nid {
				continue
			}
			srv.AddPeer(n)
		}
	}
}

func processPeerEvent(ev *p2p.PeerEvent, srv PeerAdderRemover, target int, connected map[enode.ID]struct{}, nodes map[enode.ID]*enode.Node) {
	switch ev.Type {
	case p2p.PeerEventTypeAdd:
		log.Debug("connected to a mailserver", "address", ev.Peer)
		nodeAdded(srv, ev.Peer, target, connected, nodes)
	case p2p.PeerEventTypeDrop:
		log.Debug("mailserver disconnected", "address", ev.Peer)
		nodeDisconnected(srv, ev.Peer, target, connected, nodes)
	}
}

func waitForConnections(timeout <-chan time.Time, events <-chan *p2p.PeerEvent, srv PeerAdderRemover, target int, connected map[enode.ID]struct{}, nodes map[enode.ID]*enode.Node) {
	for {
		select {
		case ev := <-events:
			processPeerEvent(ev, srv, target, connected, nodes)
			if target == len(connected) {
				return
			}
		case <-timeout:
			return
		}
	}

}
