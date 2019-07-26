// Package Node contains node logic.
package node

// @todo this is a very rough implementation that needs cleanup

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/vacp2p/mvds/peers"
	"github.com/vacp2p/mvds/protobuf"
	"github.com/vacp2p/mvds/state"
	"github.com/vacp2p/mvds/store"
	"github.com/vacp2p/mvds/transport"
)

// Mode represents the synchronization mode.
type Mode int

const (
	INTERACTIVE Mode = iota
	BATCH
)

// CalculateNextEpoch is a function used to calculate the next `SendEpoch` for a given message.
type CalculateNextEpoch func(count uint64, epoch int64) int64

// Node represents an MVDS node, it runs all the logic like sending and receiving protocol messages.
type Node struct {
	ctx    context.Context
	cancel context.CancelFunc

	store     store.MessageStore
	transport transport.Transport

	syncState state.SyncState

	peers peers.Persistence

	payloads payloads

	nextEpoch CalculateNextEpoch

	ID state.PeerID

	epoch int64
	mode  Mode
}

// NewNode returns a new node.
func NewNode(
	ms store.MessageStore,
	st transport.Transport,
	ss state.SyncState,
	nextEpoch CalculateNextEpoch,
	currentEpoch int64,
	id state.PeerID,
	mode Mode,
	pp peers.Persistence,
) *Node {
	ctx, cancel := context.WithCancel(context.Background())

	return &Node{
		ctx:       ctx,
		cancel:    cancel,
		store:     ms,
		transport: st,
		syncState: ss,
		peers:     pp,
		payloads:  newPayloads(),
		nextEpoch: nextEpoch,
		ID:        id,
		epoch:     currentEpoch,
		mode:      mode,
	}
}

// Start listens for new messages received by the node and sends out those required every epoch.
func (n *Node) Start(duration time.Duration) {
	go func() {
		for {
			select {
			case <-n.ctx.Done():
				log.Print("Watch stopped")
				return
			default:
				p := n.transport.Watch()
				go n.onPayload(p.Sender, p.Payload)
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
				time.Sleep(duration)
				err := n.sendMessages()
				if err != nil {
					log.Printf("Error sending messages: %+v\n", err)
				}
				atomic.AddInt64(&n.epoch, 1)
			}
		}
	}()
}

// Stop message reading and epoch processing
func (n *Node) Stop() {
	log.Print("Stopping node")
	n.cancel()
}

// AppendMessage sends a message to a given group.
func (n *Node) AppendMessage(groupID state.GroupID, data []byte) (state.MessageID, error) {
	m := protobuf.Message{
		GroupId:   groupID[:],
		Timestamp: time.Now().Unix(),
		Body:      data,
	}

	id := m.ID()

	peers, err := n.peers.GetByGroupID(groupID)
	if err != nil {
		return state.MessageID{}, fmt.Errorf("trying to send to unknown group %x", groupID[:4])
	}

	err = n.store.Add(&m)
	if err != nil {
		return state.MessageID{}, err
	}

	for _, p := range peers {
		t := state.OFFER
		if n.mode == BATCH {
			t = state.MESSAGE
		}

		n.insertSyncState(&groupID, id, p, t)
	}

	log.Printf("[%x] node %x sending %x\n", groupID[:4], n.ID[:4], id[:4])
	// @todo think about a way to insta trigger send messages when send was selected, we don't wanna wait for ticks here

	return id, nil
}

// RequestMessage adds a REQUEST record to the next payload for a given message ID.
func (n *Node) RequestMessage(group state.GroupID, id state.MessageID) error {
	peers, err := n.peers.GetByGroupID(group)
	if err != nil {
		return fmt.Errorf("trying to request from an unknown group %x", group[:4])
	}

	for _, p := range peers {
		exist, err := n.IsPeerInGroup(group, p)
		if err != nil {
			return err
		}

		if exist {
			continue
		}

		n.insertSyncState(&group, id, p, state.REQUEST)
	}

	return nil
}

// AddPeer adds a peer to a specific group making it a recipient of messages.
func (n *Node) AddPeer(group state.GroupID, id state.PeerID) error {
	return n.peers.Add(group, id)
}

// IsPeerInGroup checks whether a peer is in the specified group.
func (n *Node) IsPeerInGroup(g state.GroupID, p state.PeerID) (bool, error) {
	return n.peers.Exists(g, p)
}

func (n *Node) sendMessages() error {
	err := n.syncState.Map(n.epoch, func(s state.State) state.State {
		m := s.MessageID
		p := s.PeerID
		switch s.Type {
		case state.OFFER:
			n.payloads.AddOffers(p, m[:])
		case state.REQUEST:
			n.payloads.AddRequests(p, m[:])
			log.Printf("sending REQUEST (%x -> %x): %x\n", n.ID[:4], p[:4], m[:4])
		case state.MESSAGE:
			g := *s.GroupID
			//  TODO: Handle errors
			exist, err := n.IsPeerInGroup(g, p)
			if err != nil {
				return s
			}

			if !exist {
				return s
			}

			msg, err := n.store.Get(m)
			if err != nil {
				log.Printf("failed to retreive message %x %s", m[:4], err.Error())
				return s
			}

			n.payloads.AddMessages(p, msg)
			log.Printf("[%x] sending MESSAGE (%x -> %x): %x\n", g[:4], n.ID[:4], p[:4], m[:4])
		}

		return n.updateSendEpoch(s)
	})

	if err != nil {
		log.Printf("error while mapping sync state: %s", err.Error())
		return err
	}

	return n.payloads.MapAndClear(func(peer state.PeerID, payload protobuf.Payload) error {
		err := n.transport.Send(n.ID, peer, payload)
		if err != nil {
			log.Printf("error sending message: %s", err.Error())
			return err
		}
		return nil
	})

}

func (n *Node) onPayload(sender state.PeerID, payload protobuf.Payload) {
	// Acks, Requests and Offers are all arrays of bytes as protobuf doesn't allow type aliases otherwise arrays of messageIDs would be nicer.
	if err := n.onAck(sender, payload.Acks); err != nil {
		log.Printf("error processing acks: %s", err.Error())
	}
	if err := n.onRequest(sender, payload.Requests); err != nil {
		log.Printf("error processing requests: %s", err.Error())
	}
	if err := n.onOffer(sender, payload.Offers); err != nil {
		log.Printf("error processing offers: %s", err.Error())
	}
	messageIds := n.onMessages(sender, payload.Messages)
	n.payloads.AddAcks(sender, messageIds)
}

func (n *Node) onOffer(sender state.PeerID, offers [][]byte) error {
	for _, raw := range offers {
		id := toMessageID(raw)
		log.Printf("OFFER (%x -> %x): %x received.\n", sender[:4], n.ID[:4], id[:4])

		exist, err := n.store.Has(id)
		// @todo maybe ack?
		if err != nil {
			return err
		}

		if exist {
			continue
		}

		n.insertSyncState(nil, id, sender, state.REQUEST)
	}
	return nil
}

func (n *Node) onRequest(sender state.PeerID, requests [][]byte) error {
	for _, raw := range requests {
		id := toMessageID(raw)
		log.Printf("REQUEST (%x -> %x): %x received.\n", sender[:4], n.ID[:4], id[:4])

		message, err := n.store.Get(id)
		if err != nil {
			return err
		}

		if message == nil {
			log.Printf("message %x does not exist", id[:4])
			continue
		}

		groupID := toGroupID(message.GroupId)

		exist, err := n.IsPeerInGroup(groupID, sender)
		if err != nil {
			return err
		}

		if !exist {
			log.Printf("[%x] peer %x is not in group", groupID, sender[:4])
			continue
		}

		n.insertSyncState(&groupID, id, sender, state.MESSAGE)
	}

	return nil
}

func (n *Node) onAck(sender state.PeerID, acks [][]byte) error {
	for _, ack := range acks {
		id := toMessageID(ack)

		err := n.syncState.Remove(id, sender)
		if err != nil {
			log.Printf("error while removing sync state %s", err.Error())
			return err
		}

		log.Printf("ACK (%x -> %x): %x received.\n", sender[:4], n.ID[:4], id[:4])
	}
	return nil
}

func (n *Node) onMessages(sender state.PeerID, messages []*protobuf.Message) [][]byte {
	a := make([][]byte, 0)

	for _, m := range messages {
		groupID := toGroupID(m.GroupId)
		err := n.onMessage(sender, *m)
		if err != nil {
			log.Printf("Error processing messsage: %+v\n", err)
			continue
		}

		id := m.ID()
		log.Printf("[%x] sending ACK (%x -> %x): %x\n", groupID[:4], n.ID[:4], sender[:4], id[:4])
		a = append(a, id[:])
	}

	return a
}

func (n *Node) onMessage(sender state.PeerID, msg protobuf.Message) error {
	id := msg.ID()
	groupID := toGroupID(msg.GroupId)
	log.Printf("MESSAGE (%x -> %x): %x received.\n", sender[:4], n.ID[:4], id[:4])

	err := n.syncState.Remove(id, sender)
	if err != nil {
		return err
	}

	err = n.store.Add(&msg)
	if err != nil {
		return err
		// @todo process, should this function ever even have an error?
	}

	peers, err := n.peers.GetByGroupID(groupID)
	if err != nil {
		return err
	}
	for _, peer := range peers {
		if peer == sender {
			continue
		}

		n.insertSyncState(&groupID, id, peer, state.OFFER)
	}

	return nil
}

func (n *Node) insertSyncState(groupID *state.GroupID, messageID state.MessageID, peerID state.PeerID, t state.RecordType) {
	s := state.State{
		GroupID:   groupID,
		MessageID: messageID,
		PeerID:    peerID,
		Type:      t,
		SendEpoch: n.epoch + 1,
	}

	err := n.syncState.Add(s)
	if err != nil {
		log.Printf("error (%s) setting sync state group: %x id: %x peer: %x", err.Error(), groupID, messageID, peerID)
	}
}

func (n *Node) updateSendEpoch(s state.State) state.State {
	s.SendCount += 1
	s.SendEpoch += n.nextEpoch(s.SendCount, n.epoch)
	return s
}

func toMessageID(b []byte) state.MessageID {
	var id state.MessageID
	copy(id[:], b)
	return id
}

func toGroupID(b []byte) state.GroupID {
	var id state.GroupID
	copy(id[:], b)
	return id
}
