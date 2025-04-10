package leaderboard

import (
	"context"
	"sync"
)

type Signal struct {
	source int
}

func NewSignal(source int) Signal {
	return Signal{source: source}
}

func (s Signal) Source() int {
	return s.source
}

// SubscriptionManager handles event subscriptions and notifications
type SubscriptionManager struct {
	mu          sync.RWMutex
	subscribers map[chan Signal]struct{}
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscribers: make(map[chan Signal]struct{}),
	}
}

// Subscribe creates a new subscription and returns a channel that will receive notifications
func (s *SubscriptionManager) Subscribe() chan Signal {
	ch := make(chan Signal, 2) // Buffered channel to avoid blocking
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscription and closes its channel
func (s *SubscriptionManager) Unsubscribe(ch chan Signal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exist := s.subscribers[ch]
	if !exist {
		return
	}
	delete(s.subscribers, ch)
	close(ch)
}

// Emit sends a notification to all subscribers
func (s *SubscriptionManager) Emit(ctx context.Context, source int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	signal := NewSignal(source)
	for subscriber := range s.subscribers {
		select {
		case <-ctx.Done():
			// Stop sending notifications when the context is cancelled
			return
		case subscriber <- signal:
			// Notified successfully
		default:
			// Skip notification if the subscriber's channel is full (non-blocking)
		}
	}
}
