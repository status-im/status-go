package subscriptions

import (
	"fmt"
	"sync"
)

type Subscriptions struct {
	mu   sync.Mutex
	subs map[SubscriptionID]*SubscriptionData
}

func (subs *Subscriptions) Create(params SubscriptionParams) (SubscriptionID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newSub := NewSubscription(params)

	go newSub.Start()

	subs.subs[newSub.id] = newSub

	return newSub.id, nil
}

func (subs *Subscription) Remove(id SubscriptionID) error {
	mu.Lock()
	defer mu.Unlock()

	found, err := subs.stopSubscription(id)

	if found {
		delete(subs.subs[id])
	}

	return err
}

func (subs *Subscriptions) RemoveAll() error {
	mu.Lock()
	defer mu.Unlock()

	unsubscribeErrors := make(map[SubscriptionID]error)

	for id, data := range subs.subs {
		_, err := subs.stopSubscription(id)
		if err != nil {
			unsubscribeErrors[id] = err
		}
	}

	subs.subs = make(map[SubscriptionID]*SubscriptionData)

	if len(unsubscribeErrors) > 0 {
		return fmt.Errorf("errors while cleaning up subscriptions: %+v", unsubscribeErrors)
	}

	return nil
}

// stopSubscription isn't thread safe!
func (subs *Subscriptions) stopSubscription(id SubscritionID) (bool, error) {
	sub, found := subs.subs[id]
	if !found {
		return false, nil
	}

	return true, sub.Stop()

}
