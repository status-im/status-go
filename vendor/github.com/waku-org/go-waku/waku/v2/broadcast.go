package v2

import (
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

// Adapted from https://github.com/dustin/go-broadcast/commit/f664265f5a662fb4d1df7f3533b1e8d0e0277120
// by Dustin Sallings (c) 2013, which was released under MIT license

type doneCh chan struct{}

type chOperation struct {
	ch    chan<- *protocol.Envelope
	topic *string
	done  doneCh
}

type broadcastOutputs map[chan<- *protocol.Envelope]struct{}

type broadcaster struct {
	input chan *protocol.Envelope
	reg   chan chOperation
	unreg chan chOperation

	outputs         broadcastOutputs
	outputsPerTopic map[string]broadcastOutputs
}

// The Broadcaster interface describes the main entry points to
// broadcasters.
type Broadcaster interface {
	// Register a new channel to receive broadcasts from a pubsubtopic
	Register(topic *string, newch chan<- *protocol.Envelope)
	// Register a new channel to receive broadcasts from a pubsub topic and return a channel to wait until this operation is complete
	WaitRegister(topic *string, newch chan<- *protocol.Envelope) doneCh
	// Unregister a channel so that it no longer receives broadcasts from a pubsub topic
	Unregister(topic *string, newch chan<- *protocol.Envelope)
	// Unregister a subscriptor channel and return a channel to wait until this operation is done
	WaitUnregister(topic *string, newch chan<- *protocol.Envelope) doneCh
	// Shut this broadcaster down.
	Close()
	// Submit a new object to all subscribers
	Submit(*protocol.Envelope)
}

func (b *broadcaster) broadcast(m *protocol.Envelope) {
	for ch := range b.outputs {
		ch <- m
	}

	outputs, ok := b.outputsPerTopic[m.PubsubTopic()]
	if !ok {
		return
	}

	for ch := range outputs {
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
				if broadcastee.topic != nil {
					topicOutputs, ok := b.outputsPerTopic[*broadcastee.topic]
					if !ok {
						b.outputsPerTopic[*broadcastee.topic] = make(broadcastOutputs)
						topicOutputs = b.outputsPerTopic[*broadcastee.topic]
					}

					topicOutputs[broadcastee.ch] = struct{}{}
					b.outputsPerTopic[*broadcastee.topic] = topicOutputs
				} else {
					b.outputs[broadcastee.ch] = struct{}{}
				}
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
			if broadcastee.topic != nil {
				topicOutputs, ok := b.outputsPerTopic[*broadcastee.topic]
				if !ok {
					continue
				}
				delete(topicOutputs, broadcastee.ch)
				b.outputsPerTopic[*broadcastee.topic] = topicOutputs
			} else {
				delete(b.outputs, broadcastee.ch)
			}

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
		input:           make(chan *protocol.Envelope, buflen),
		reg:             make(chan chOperation),
		unreg:           make(chan chOperation),
		outputs:         make(broadcastOutputs),
		outputsPerTopic: make(map[string]broadcastOutputs),
	}

	go b.run()

	return b
}

// Register a subscriptor channel and return a channel to wait until this operation is done
func (b *broadcaster) WaitRegister(topic *string, newch chan<- *protocol.Envelope) doneCh {
	d := make(doneCh)
	b.reg <- chOperation{
		ch:    newch,
		topic: topic,
		done:  d,
	}
	return d
}

// Register a subscriptor channel
func (b *broadcaster) Register(topic *string, newch chan<- *protocol.Envelope) {
	b.reg <- chOperation{
		ch:    newch,
		topic: topic,
		done:  nil,
	}
}

// Unregister a subscriptor channel and return a channel to wait until this operation is done
func (b *broadcaster) WaitUnregister(topic *string, newch chan<- *protocol.Envelope) doneCh {
	d := make(doneCh)
	b.unreg <- chOperation{
		ch:    newch,
		topic: topic,
		done:  d,
	}
	return d
}

// Unregister a subscriptor channel
func (b *broadcaster) Unregister(topic *string, newch chan<- *protocol.Envelope) {
	b.unreg <- chOperation{
		ch:    newch,
		topic: topic,
		done:  nil,
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
