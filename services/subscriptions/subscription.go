package subscriptions

import (
	"fmt"
	"time"
)

type SubscriptionID string

type Subscription struct {
	id     SubscriptionID
	signal *filterSignal
	quit   chan interface{}
	filter filter
}

func NewSubscription(namespace string, filter filter) *Subscription {
	subscriptionID := NewSubscriptionID(namespace, filter.getID())

	quit := make(chan interface{})

	return &Subscription{
		id:     subscriptionID,
		quit:   quit,
		signal: newFilterSignal(string(subscriptionID)),
		filter: filter,
	}
}

func (s *Subscription) Start(checkPeriod time.Duration) {
	ticker := time.NewTicker(checkPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			filterData, err := s.filter.getChanges()
			if err != nil {
				s.signal.SendError(err)
			} else if filterData != nil {
				s.signal.SendData(filterData)
			}
		case <-s.quit:
			return
		}
	}
}

func (s *Subscription) Stop() error {
	close(s.quit)
	return s.filter.uninstall()
}

func NewSubscriptionID(namespace, filterID string) SubscriptionID {
	return SubscriptionID(fmt.Sprintf("%s-%s", namespace, filterID))
}
