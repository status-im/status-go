// Package Node contains node logic.
package node

// @todo this is a very rough implementation that needs cleanup

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
	"github.com/status-im/mvds/store"
	"github.com/status-im/mvds/transport"
)

// Mode represents the synchronization mode.
type Mode int

const (
	INTERACTIVE Mode = iota
	BATCH
)

type calculateNextEpoch func(count uint64, epoch int64) int64

// Node represents an MVDS node, it runs all the logic like sending and receiving protocol messages.
type Node struct {
	ctx    context.Context
	cancel context.CancelFunc

	store     store.MessageStore
	transport transport.Transport

	syncState state.SyncState

	peers     map[state.GroupID][]state.PeerID

	payloads payloads

	nextEpoch calculateNextEpoch

	ID state.PeerID

	epoch int64
	mode  Mode
}

// NewNode returns a new node.
func NewNode(
	ms store.MessageStore,
	st transport.Transport,
	ss state.SyncState,
	nextEpoch calculateNextEpoch,
	currentEpoch int64,
	id state.PeerID,
	mode Mode,
) *Node {
	ctx, cancel := context.WithCancel(context.Background())

	return &Node{
		ctx:       ctx,
		cancel:    cancel,
		store:     ms,
		transport: st,
		syncState: ss,
		peers:     make(map[state.GroupID][]state.PeerID),
		payloads:  newPayloads(),
		nextEpoch: nextEpoch,
		ID:        id,
		epoch:     currentEpoch,
		mode:      mode,
	}
}

// Start listens for new messages received by the node and sends out those required every epoch.
func (n *Node) Start() {
	go func() {
		for {
			select {
			case <-n.ctx.Done():
				log.Print("Watch stopped")
				return
			default:
				p := n.transport.Watch()
				go n.onPayload(p.Group, p.Sender, p.Payload)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-n.ctx.Done():
				log.Print("Epoch processing stopped")
				return
			default:
				log.Printf("Node: %x Epoch: %d", n.ID[:4], n.epoch)
				time.Sleep(1 * time.Second)

				n.sendMessages()
				atomic.AddInt64(&n.epoch, 1)
			}
		}
	}()
}

// Stop message reading and epoch processing
func (n *Node) Stop() {
	n.cancel()
}

// AppendMessage sends a message to a given group.
func (n *Node) AppendMessage(group state.GroupID, data []byte) (state.MessageID, error) {
	m := protobuf.Message{
		GroupId:   group[:],
		Timestamp: time.Now().Unix(),
		Body:      data,
	}

	id := state.ID(m)

	peers, ok := n.peers[group]
	if !ok {
		return state.MessageID{}, fmt.Errorf("trying to send to unknown group %x", group[:4])
	}

	err := n.store.Add(m)
	if err != nil {
		return state.MessageID{}, err
	}

	go func() {
		for _, p := range peers {
			if !n.IsPeerInGroup(group, p) {
				continue
			}

			t := state.OFFER
			if n.mode == BATCH {
				t = state.MESSAGE
			}

			n.insertSyncState(group, id, p, t)
		}
	}()

	log.Printf("[%x] node %x sending %x\n", group[:4], n.ID[:4], id[:4])
	// @todo think about a way to insta trigger send messages when send was selected, we don't wanna wait for ticks here

	return id, nil
}

// AddPeer adds a peer to a specific group making it a recipient of messages.
func (n *Node) AddPeer(group state.GroupID, id state.PeerID) {
	if _, ok := n.peers[group]; !ok {
		n.peers[group] = make([]state.PeerID, 0)
	}

	n.peers[group] = append(n.peers[group], id)
}

// IsPeerInGroup checks whether a peer is in the specified group.
func (n Node) IsPeerInGroup(g state.GroupID, p state.PeerID) bool {
	for _, peer := range n.peers[g] {
		if bytes.Equal(peer[:], p[:]) {
			return true
		}
	}

	return false
}

func (n *Node) sendMessages() {
	err := n.syncState.Map(n.epoch, func(g state.GroupID, m state.MessageID, p state.PeerID, s state.State) state.State {
		if !n.IsPeerInGroup(g, p) {
			return s
		}

		switch s.Type {
		case state.OFFER:
			n.payloads.AddOffers(g, p, m[:])
		case state.REQUEST:
			n.payloads.AddRequests(g, p, m[:])
			log.Printf("[%x] sending REQUEST (%x -> %x): %x\n", g[:4], n.ID[:4], p[:4], m[:4])
		case state.MESSAGE:
			msg, err := n.store.Get(m)
			if err != nil {
				log.Printf("failed to retreive message %x %s", m[:4], err.Error())
				return s
			}

			n.payloads.AddMessages(g, p, &msg)
			log.Printf("[%x] sending MESSAGE (%x -> %x): %x\n", g[:4], n.ID[:4], p[:4], m[:4])
		}

		return n.updateSendEpoch(s)
	})

	if err != nil {
		log.Printf("error while mapping sync state: %s", err.Error())
	}

	n.payloads.MapAndClear(func(id state.GroupID, peer state.PeerID, payload protobuf.Payload) {
		err := n.transport.Send(id, n.ID, peer, payload)
		if err != nil {
			log.Printf("error sending message: %s", err.Error())
			//	@todo
		}
	})
}

func (n *Node) onPayload(group state.GroupID, sender state.PeerID, payload protobuf.Payload) {
	// Acks, Requests and Offers are all arrays of bytes as protobuf doesn't allow type aliases otherwise arrays of messageIDs would be nicer.
	n.onAck(group, sender, payload.Acks)
	n.onRequest(group, sender, payload.Requests)
	n.onOffer(group, sender, payload.Offers)
	n.payloads.AddAcks(group, sender, n.onMessages(group, sender, payload.Messages)...)
}

func (n *Node) onOffer(group state.GroupID, sender state.PeerID, offers [][]byte) {
	for _, raw := range offers {
		id := toMessageID(raw)
		log.Printf("[%x] OFFER (%x -> %x): %x received.\n", group[:4], sender[:4], n.ID[:4], id[:4])

		// @todo maybe ack?
		if n.store.Has(id) {
			continue
		}

		n.insertSyncState(group, id, sender, state.REQUEST)
	}
}

func (n *Node) onRequest(group state.GroupID, sender state.PeerID, requests [][]byte) {
	for _, raw := range requests {
		id := toMessageID(raw)
		log.Printf("[%x] REQUEST (%x -> %x): %x received.\n", group[:4], sender[:4], n.ID[:4], id[:4])

		if !n.IsPeerInGroup(group, sender) {
			log.Printf("[%x] peer %x is not in group", group[:4], sender[:4])
			continue
		}

		if !n.store.Has(id) {
			log.Printf("message %x does not exist", id[:4])
			continue
		}

		n.insertSyncState(group, id, sender, state.MESSAGE)
	}
}

func (n *Node) onAck(group state.GroupID, sender state.PeerID, acks [][]byte) {
	for _, raw := range acks {
		id := toMessageID(raw)

		err := n.syncState.Remove(group, id, sender)
		if err != nil {
			log.Printf("error while removing sync state %s", err.Error())
			continue
		}

		log.Printf("[%x] ACK (%x -> %x): %x received.\n", group[:4], sender[:4], n.ID[:4], id[:4])
	}
}

func (n *Node) onMessages(group state.GroupID, sender state.PeerID, messages []*protobuf.Message) [][]byte {
	a := make([][]byte, 0)

	for _, m := range messages {
		err := n.onMessage(group, sender, *m)
		if err != nil {
			// @todo
			continue
		}

		id := state.ID(*m)
		log.Printf("[%x] sending ACK (%x -> %x): %x\n", group[:4], n.ID[:4], sender[:4], id[:4])
		a = append(a, id[:])
	}

	return a
}

func (n *Node) onMessage(group state.GroupID, sender state.PeerID, msg protobuf.Message) error {
	id := state.ID(msg)
	log.Printf("[%x] MESSAGE (%x -> %x): %x received.\n", group[:4], sender[:4], n.ID[:4], id[:4])

	err := n.syncState.Remove(group, id, sender)
	if err != nil {
		return err
	}

	go func() {
		for _, peer := range n.peers[group] {
			if peer == sender {
				continue
			}

			n.insertSyncState(group, id, peer, state.OFFER)
		}
	}()

	err = n.store.Add(msg)
	if err != nil {
		return err
		// @todo process, should this function ever even have an error?
	}

	return nil
}

func (n *Node) insertSyncState(group state.GroupID, id state.MessageID, p state.PeerID, t state.RecordType) {
	s := state.State{
		Type: t,
		SendEpoch: n.epoch + 1,
	}

	err := n.syncState.Set(group, id, p, s)
	if err != nil {
		log.Printf("error (%s) setting sync state group: %x id: %x peer: %x", err.Error(), group[:4], id[:4], p[:4])
	}
}

func (n Node) updateSendEpoch(s state.State) state.State {
	s.SendCount += 1
	s.SendEpoch += n.nextEpoch(s.SendCount, n.epoch)
	return s
}

func toMessageID(b []byte) state.MessageID {
	var id state.MessageID
	copy(id[:], b)
	return id
}
