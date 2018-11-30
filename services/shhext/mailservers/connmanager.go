package mailservers

import (
	"sync"

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

//EnvelopeEventScubriber interface to subscribe for whisper.EnvelopeEvent's.
type EnvelopeEventScubriber interface {
	SubscribeEnvelopeEvents(chan<- whisperv6.EnvelopeEvent) event.Subscription
}

type p2pServer interface {
	PeerAdderRemover
	PeerEventsSubscriber
}

// NewConnectionManager creates an instance of ConnectionManager.
func NewConnectionManager(server p2pServer, whisper EnvelopeEventScubriber, target int) *ConnectionManager {
	return &ConnectionManager{
		server:          server,
		whisper:         whisper,
		connectedTarget: target,
		notifications:   make(chan []*enode.Node),
	}
}

// ConnectionManager manages keeps target of peers connected.
type ConnectionManager struct {
	wg   sync.WaitGroup
	quit chan struct{}

	server  p2pServer
	whisper EnvelopeEventScubriber

	notifications   chan []*enode.Node
	connectedTarget int
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
		for {
			select {
			case <-ps.quit:
				sub.Unsubscribe()
				whisperSub.Unsubscribe()
				ps.wg.Done()
				return
			case err := <-sub.Err():
				log.Error("retry after error to subscribe for p2p events", "error", err)
				sub = ps.server.SubscribeEvents(events)
			case err := <-whisperSub.Err():
				log.Error("retry after error to subscribe for whisper events", "error", err)
				whisperSub = ps.whisper.SubscribeEnvelopeEvents(whisperEvents)
			case newNodes := <-ps.notifications:
				replace := map[enode.ID]*enode.Node{}
				for _, n := range newNodes {
					replace[n.ID()] = n
				}
				replaceNodes(ps.server, ps.connectedTarget, connected, current, replace)
				current = replace
			case ev := <-events:
				switch ev.Type {
				case p2p.PeerEventTypeAdd:
					log.Debug("connected to a mailserver", "address", ev.Peer)
					nodeAdded(ps.server, ev.Peer, ps.connectedTarget, connected, current)
				case p2p.PeerEventTypeDrop:
					log.Debug("mailserver disconnected", "address", ev.Peer)
					nodeDisconnected(ps.server, ev.Peer, ps.connectedTarget, connected, current)
				}
			case ev := <-whisperEvents:
				switch ev.Event {
				case whisperv6.EventMailServerRequestExpired:
					log.Debug("request to a mail server expired, disconnet a peer", "address", ev.Peer)
					nodeDisconnected(ps.server, ev.Peer, ps.connectedTarget, connected, current)
				}
				// TODO what about completed but with error? what about expired envelopes?
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
	n, exist := nodes[peer] // not related event
	if !exist {
		return
	}
	_, exist = connected[peer] // already removed
	if !exist {
		return
	}
	if len(nodes) == 1 { // keep node connected if we don't have another choice
		return
	}
	srv.RemovePeer(n) // remove peer permanently, otherwise p2p.Server will try to reconnect
	delete(connected, peer)
	if len(connected) < target { // try to connect with any other selected, but not connected node
		for nid, n := range nodes {
			_, exist := connected[nid]
			if exist || peer == nid {
				continue
			}
			srv.AddPeer(n)
		}
	}
}
