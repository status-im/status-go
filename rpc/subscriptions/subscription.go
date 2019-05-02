package subscriptions

import (
	"fmt"
	"time"
)

type SubscriptionID string

type cleanupFunc func() error
type monitorFunc func() ([]string, error)

type Subscription struct {
	id          string
	signal      *filterSignal
	quit        chan interface{}
	cleanupFunc cleanupFunc
	monitorFunc monitorFunc
}

type SubscriptionParams struct {
	namespace   string
	filterID    string
	cleanupFunc cleanupFunc
	monitorFunc monitorFunc
}

func NewSubscription(params SubscriptionParams) *Subscription {
	subscriptionID := NewSubscriptionID(params.namespace, params.filterID)
	quit := make(chan interface{})

	return &Subscription{
		id:          subscriptionID,
		quit:        quit,
		signal:      newFilterSignal(subscriptionID),
		cleanupFunc: params.cleanupFunc,
		monitorFunc: params.monitorFunc,
	}
}

func (s *Subscription) Start() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			filterData, err := s.monitorFunc()
			if err != nil {
				s.signal.SendError(err)
			} else {
				s.signal.SendData(filterData)
			}
		case <-s.quit:
			return
		}
	}
}

func (s *Subscription) Stop() error {
	close(s.quit)
	return s.cleanupFunc()
}

func NewSubscriptionID(namespace, filterID string) SubscriptionID {
	return SubscriptionID(fmt.Sprintf("%s-%s", namespace, filterID))
}
