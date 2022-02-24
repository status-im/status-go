package v2

import (
	"github.com/status-im/go-waku/waku/v2/protocol"
)

// Adapted from https://github.com/dustin/go-broadcast/commit/f664265f5a662fb4d1df7f3533b1e8d0e0277120
// by Dustin Sallings (c) 2013, which was released under MIT license

type doneCh chan struct{}

type chOperation struct {
	ch   chan<- *protocol.Envelope
	done doneCh
}

type broadcaster struct {
	input chan *protocol.Envelope
	reg   chan chOperation
	unreg chan chOperation

	outputs map[chan<- *protocol.Envelope]bool
}

// The Broadcaster interface describes the main entry points to
// broadcasters.
type Broadcaster interface {
	// Register a new channel to receive broadcasts
	Register(chan<- *protocol.Envelope)
	// Register a new channel to receive broadcasts and return a channel to wait until this operation is complete
	WaitRegister(newch chan<- *protocol.Envelope) doneCh
	// Unregister a channel so that it no longer receives broadcasts.
	Unregister(chan<- *protocol.Envelope)
	// Unregister a subscriptor channel and return a channel to wait until this operation is done
	WaitUnregister(newch chan<- *protocol.Envelope) doneCh
	// Shut this broadcaster down.
	Close()
	// Submit a new object to all subscribers
	Submit(*protocol.Envelope)
}

func (b *broadcaster) broadcast(m *protocol.Envelope) {
	for ch := range b.outputs {
		ch <- m
	}
}

func (b *broadcaster) run() {
	for {
		select {
		case m := <-b.input:
			b.broadcast(m)
		case broadcastee, ok := <-b.reg:
			if ok {
				b.outputs[broadcastee.ch] = true
				if broadcastee.done != nil {
					broadcastee.done <- struct{}{}
				}
			} else {
				if broadcastee.done != nil {
					broadcastee.done <- struct{}{}
				}
				return
			}
		case broadcastee := <-b.unreg:
			delete(b.outputs, broadcastee.ch)
			if broadcastee.done != nil {
				broadcastee.done <- struct{}{}
			}
		}
	}
}

// NewBroadcaster creates a Broadcaster with an specified length
// It's used to register subscriptors that will need to receive
// an Envelope containing a WakuMessage
func NewBroadcaster(buflen int) Broadcaster {
	b := &broadcaster{
		input:   make(chan *protocol.Envelope, buflen),
		reg:     make(chan chOperation),
		unreg:   make(chan chOperation),
		outputs: make(map[chan<- *protocol.Envelope]bool),
	}

	go b.run()

	return b
}

// Register a subscriptor channel and return a channel to wait until this operation is done
func (b *broadcaster) WaitRegister(newch chan<- *protocol.Envelope) doneCh {
	d := make(doneCh)
	b.reg <- chOperation{
		ch:   newch,
		done: d,
	}
	return d
}

// Register a subscriptor channel
func (b *broadcaster) Register(newch chan<- *protocol.Envelope) {
	b.reg <- chOperation{
		ch:   newch,
		done: nil,
	}
}

// Unregister a subscriptor channel and return a channel to wait until this operation is done
func (b *broadcaster) WaitUnregister(newch chan<- *protocol.Envelope) doneCh {
	d := make(doneCh)
	b.unreg <- chOperation{
		ch:   newch,
		done: d,
	}
	return d
}

// Unregister a subscriptor channel
func (b *broadcaster) Unregister(newch chan<- *protocol.Envelope) {
	b.unreg <- chOperation{
		ch:   newch,
		done: nil,
	}
}

// Closes the broadcaster. Used to stop receiving new subscribers
func (b *broadcaster) Close() {
	close(b.reg)
}

// Submits an Envelope to be broadcasted among all registered subscriber channels
func (b *broadcaster) Submit(m *protocol.Envelope) {
	if b != nil {
		b.input <- m
	}
}
