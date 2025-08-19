package pubsub

import (
	"sync"
)

// subscription represents a subscription to a publisher.
type subscription[T any] struct {
	mu sync.RWMutex // protects the channel from being closed while a value is being published to it
	ch chan T
}

func newSubscription[T any](bufferSize uint) *subscription[T] {
	return &subscription[T]{
		ch: make(chan T, bufferSize),
	}
}

// Send the value val to the channel if it is not full. If the channel is busy,
// the event is dropped and the subscriber won't be notified.
// The channel is protected from being closed while a value is being sent to it.
func (s *subscription[T]) Send(val T) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case s.ch <- val:
	default:
		// Drop event if buffer is full
	}
}

// Close the channel, ensuring no value is being sent to it
func (s *subscription[T]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	close(s.ch)
}

type UnsubFn = func()

// TypePublisher manages subscriptions to events of a single type.
type TypePublisher[T any] struct {
	mu     sync.RWMutex // protects the subs map and closed flag
	subs   map[chan T]*subscription[T]
	closed bool
}

func NewTypePublisher[T any]() *TypePublisher[T] {
	return &TypePublisher[T]{
		subs:   make(map[chan T]*subscription[T]),
		closed: false,
	}
}

func (p *TypePublisher[T]) Publish(val T) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return
	}

	for _, sub := range p.subs {
		sub.Send(val)
	}
}

func (p *TypePublisher[T]) Subscribe(bufferSize uint) chan T {
	p.mu.Lock()
	defer p.mu.Unlock()

	sub := newSubscription[T](bufferSize)
	p.subs[sub.ch] = sub
	return sub.ch
}

func (p *TypePublisher[T]) Unsubscribe(ch chan T) {
	p.mu.Lock()
	defer p.mu.Unlock()

	sub, ok := p.subs[ch]
	if !ok {
		return
	}
	delete(p.subs, ch)
	sub.Close()
}

func (p *TypePublisher[T]) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	subs := p.subs
	for _, sub := range subs {
		sub.Close()
	}
	p.subs = make(map[chan T]*subscription[T])
}

func (p *TypePublisher[T]) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.closed
}
