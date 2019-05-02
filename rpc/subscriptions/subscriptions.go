package subscriptions

import (
	"fmt"
	"sync"
	"time"
)

type SubscriptionID string

type cleanupFunc func() error

type SubscriptionData struct {
	quit        chan interface{}
	cleanupFunc cleanupFunc
}

type Subscriptions struct {
	mu   sync.Mutex
	subs map[SubscriptionID]*SubscriptionData
}

func (s *Subscriptions) Add(namespace, filterID string, cleanup cleanupFunc) (SubscriptionID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscriptionID := NewSubscriptionID(namespace, filterID)

	quit := make(chan interface{})
	go monitorFilter(quit)

	s.subs[subscriptionID] = SubscriptionData{
		quit:        quit,
		cleanupFunc: cleanup,
	}

	return subscriptionID, nil
}

func (s *Subscription) Remove(id SubscriptionID) error {
	mu.Lock()
	defer mu.Unlock()

	data, found := s.subs[id]
	if !found {
		return nil
	}

	delete(s.subs[id])

	return s.stopSubscription(id)
}

func (s *Subscriptions) RemoveAll() error {
	mu.Lock()
	defer mu.Unlock()

	unsubscribeErrors := make(map[SubscriptionID]error)

	for id, data := range s.subs {
		err := s.stopSubscription(id)
		if err != nil {
			unsubscribeErrors[id] = err
		}
	}

	s.subs = make(map[SubscriptionID]*SubscriptionData)

	if len(unsubscribeErrors) > 0 {
		return fmt.Errorf("errors while cleaning up subscriptions: %+v", unsubscribeErrors)
	}

	return nil
}

func (s *Subscription) stopSubscription(id SubscritionID) error {
	data, found := s.subs[id]
	if !found {
		return nil
	}

	close(data.quit)
	return data.cleanupFunc()
}

func monitorFilter(quit chan interface{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// check for new stuff and report signal
		case <-quit:
			return
		}
	}
}

func NewSubscriptionID(namespace, filterID string) SubscriptionID {
	return SubscriptionID(fmt.Sprintf("%s-%s", namespace, filterID))
}
