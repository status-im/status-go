package node

import (
	"sync"

	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
)

type payloads struct {
	sync.Mutex

	payloads map[state.GroupID]map[state.PeerID]protobuf.Payload
}
// @todo check in all the functions below that we aren't duplicating stuff

func newPayloads() payloads {
	return payloads{
		payloads: make(map[state.GroupID]map[state.PeerID]protobuf.Payload),
	}
}

func (p *payloads) AddOffers(group state.GroupID, peer state.PeerID, offers ...[]byte) {
	p.Lock()
	defer p.Unlock()

	payload := p.get(group, peer)
	if payload.Offer == nil {
		payload.Offer = &protobuf.Offer{Id: make([][]byte, 0)}
	}

	payload.Offer.Id = append(payload.Offer.Id, offers...)

	p.set(group, peer, payload)
}

func (p *payloads) AddAcks(group state.GroupID, peer state.PeerID, acks ...[]byte) {
	p.Lock()
	defer p.Unlock()

	payload := p.get(group, peer)
	if payload.Ack == nil {
		payload.Ack = &protobuf.Ack{Id: make([][]byte, 0)}
	}

	payload.Ack.Id = append(payload.Ack.Id, acks...)

	p.set(group, peer, payload)
}

func (p *payloads) AddRequests(group state.GroupID, peer state.PeerID, request ...[]byte) {
	p.Lock()
	defer p.Unlock()

	payload := p.get(group, peer)
	if payload.Request == nil {
		payload.Request = &protobuf.Request{Id: make([][]byte, 0)}
	}

	payload.Request.Id = append(payload.Request.Id, request...)

	p.set(group, peer, payload)
}

func (p *payloads) AddMessages(group state.GroupID, peer state.PeerID, messages ...*protobuf.Message) {
	p.Lock()
	defer p.Unlock()

	payload := p.get(group, peer)
	if payload.Messages == nil {
		payload.Messages = make([]*protobuf.Message, 0)
	}

	payload.Messages = append(payload.Messages, messages...)
	p.set(group, peer, payload)
}

func (p *payloads) MapAndClear(f func(state.GroupID, state.PeerID, protobuf.Payload)) {
	p.Lock()
	defer p.Unlock()

	for g, payloads := range p.payloads {
		for peer, payload := range payloads {
			f(g, peer, payload)
		}
	}

	p.payloads = make(map[state.GroupID]map[state.PeerID]protobuf.Payload)
}

func (p *payloads) get(id state.GroupID, peer state.PeerID) protobuf.Payload {
	payload, _ := p.payloads[id][peer]
	return payload
}

func (p *payloads) set(id state.GroupID, peer state.PeerID, payload protobuf.Payload) {
	_, ok := p.payloads[id]
	if !ok {
		p.payloads[id] = make(map[state.PeerID]protobuf.Payload)
	}

	p.payloads[id][peer] = payload
}
