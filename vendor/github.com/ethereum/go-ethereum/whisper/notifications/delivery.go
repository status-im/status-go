package notifications

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/message"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

// DeliveryState holds references to a message delivered through the network.
type DeliveryState struct {
	IsP2P bool
	P2P   *whisper.P2PMessageState
	RPC   *whisper.RPCMessageState
}

// DeliverySubscriber defines a function type for subscrubers.
type DeliverySubscriber func(DeliveryState)

// P2PDeliverySubscriber defines a function type for receiving p2p message status.
type P2PDeliverySubscriber func(*whisper.P2PMessageState)

// RPCDeliverySubscriber defines a function type for receiving rpc message status.
type RPCDeliverySubscriber func(*whisper.RPCMessageState)

// DeliveryService defines a notification implementation for listening to message status
// events.
type DeliveryService struct {
	sml  sync.RWMutex
	subs []DeliverySubscriber
}

// SendP2PState delivers a rpc message status to all subscribers.
func (d *DeliveryService) SendP2PState(state whisper.P2PMessageState) {
	if state.Timestamp.IsZero() {
		state.Timestamp = time.Now()
	}
	d.sendState(DeliveryState{P2P: &state})
}

// SendRPCState delivers a rpc message status to all subscribers.
func (d *DeliveryService) SendRPCState(state whisper.RPCMessageState) {
	if state.Timestamp.IsZero() {
		state.Timestamp = time.Now()
	}
	d.sendState(DeliveryState{RPC: &state})
}

// SendState delivers envelope with status to all subscribers.
func (d *DeliveryService) sendState(mds DeliveryState) {
	d.sml.RLock()
	defer d.sml.RUnlock()

	for _, item := range d.subs {
		item(mds)
	}
}

// Unsubscribe removes subscriber into delivery subscription list.
func (d *DeliveryService) Unsubscribe(ind int) {
	d.sml.Lock()
	defer d.sml.Unlock()

	if ind > -1 && ind < len(d.subs) {
		d.subs = append(d.subs[:ind], d.subs[ind+1:]...)
	}
}

// SubscribeForP2P delivers rpc messages status events to the callback.
func (d *DeliveryService) SubscribeForP2P(sub P2PDeliverySubscriber) int {
	return d.Subscribe(func(m DeliveryState) {
		if m.IsP2P || m.P2P == nil {
			return
		}

		sub(m.P2P)
	})
}

// SubscribeForRPC delivers rpc messages status events to the callback.
func (d *DeliveryService) SubscribeForRPC(sub RPCDeliverySubscriber) int {
	return d.Subscribe(func(m DeliveryState) {
		if m.IsP2P || m.RPC == nil {
			return
		}

		sub(m.RPC)
	})
}

// ByP2PDirection filters out p2p messages status events who status does not match provided.
func (d *DeliveryService) ByP2PDirection(dir message.Direction, sub P2PDeliverySubscriber) int {
	return d.SubscribeForP2P(func(msg *whisper.P2PMessageState) {
		if msg.Direction == dir {
			sub(msg)
		}
	})
}

// ByRPCDirection filters out prc messages status events who status does not match provided.
func (d *DeliveryService) ByRPCDirection(dir message.Direction, sub RPCDeliverySubscriber) int {
	return d.SubscribeForRPC(func(msg *whisper.RPCMessageState) {
		if msg.Direction == dir {
			sub(msg)
		}
	})
}

// ByP2PStatus filters out p2p messages status events who status does not match provided.
func (d *DeliveryService) ByP2PStatus(status message.Status, sub P2PDeliverySubscriber) int {
	return d.SubscribeForP2P(func(msg *whisper.P2PMessageState) {
		if msg.Status == status {
			sub(msg)
		}
	})
}

// ByRPCStatus filters out prc messages status events who status does not match provided.
func (d *DeliveryService) ByRPCStatus(status message.Status, sub RPCDeliverySubscriber) int {
	return d.SubscribeForRPC(func(msg *whisper.RPCMessageState) {
		if msg.Status == status {
			sub(msg)
		}
	})
}

// Subscribe adds subscriber into delivery subscription list.
// It returns the index of subscription.
func (d *DeliveryService) Subscribe(sub DeliverySubscriber) int {
	d.sml.Lock()
	defer d.sml.Unlock()

	d.subs = append(d.subs, sub)
	return len(d.subs)
}
