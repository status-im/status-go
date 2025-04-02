//go:build gowaku_no_rln
// +build gowaku_no_rln

package leaderboard

import (
	"context"
	"sync"
)

// SubscriptionManager handles event subscriptions and notifications
type SubscriptionManager struct {
	mu          sync.RWMutex
	subscribers map[chan struct{}]struct{}
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscribers: make(map[chan struct{}]struct{}),
	}
}

// Subscribe creates a new subscription and returns a channel that will receive notifications
func (s *SubscriptionManager) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscription and closes its channel
func (s *SubscriptionManager) Unsubscribe(ch chan struct{}) {
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
func (s *SubscriptionManager) Emit(ctx context.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for subscriber := range s.subscribers {
		select {
		case <-ctx.Done():
			// Stop sending notifications when the context is cancelled
			return
		case subscriber <- struct{}{}:
			// Notified successfully
		default:
			// Skip notification if the subscriber's channel is full (non-blocking)
		}
	}
}
