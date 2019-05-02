package subscriptions

import (
	"fmt"
	"sync"
)

type Subscriptions struct {
	mu   sync.Mutex
	subs map[SubscriptionID]*Subscription
}

func NewSubscriptions() *Subscriptions {
	return &Subscriptions{
		mu:   sync.Mutex{},
		subs: make(map[SubscriptionID]*Subscription),
	}
}

func (subs *Subscriptions) Create(namespace string, filter filter) (SubscriptionID, error) {
	subs.mu.Lock()
	defer subs.mu.Unlock()

	newSub := NewSubscription(namespace, filter)

	go newSub.Start()

	subs.subs[newSub.id] = newSub

	return newSub.id, nil
}

func (subs *Subscriptions) Remove(id SubscriptionID) error {
	subs.mu.Lock()
	defer subs.mu.Unlock()

	found, err := subs.stopSubscription(id)

	if found {
		delete(subs.subs, id)
	}

	return err
}

func (subs *Subscriptions) RemoveAll() error {
	subs.mu.Lock()
	defer subs.mu.Unlock()

	unsubscribeErrors := make(map[SubscriptionID]error)

	for id, _ := range subs.subs {
		_, err := subs.stopSubscription(id)
		if err != nil {
			unsubscribeErrors[id] = err
		}
	}

	subs.subs = make(map[SubscriptionID]*Subscription)

	if len(unsubscribeErrors) > 0 {
		return fmt.Errorf("errors while cleaning up subscriptions: %+v", unsubscribeErrors)
	}

	return nil
}

// stopSubscription isn't thread safe!
func (subs *Subscriptions) stopSubscription(id SubscriptionID) (bool, error) {
	sub, found := subs.subs[id]
	if !found {
		return false, nil
	}

	return true, sub.Stop()

}
