package healthmanager

import (
	"context"
	"sync"
)

type SubscriptionManager struct {
	mu          sync.RWMutex
	subscribers map[chan struct{}]struct{}
}

func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscribers: make(map[chan struct{}]struct{}),
	}
}

func (s *SubscriptionManager) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[ch] = struct{}{}
	return ch
}

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
