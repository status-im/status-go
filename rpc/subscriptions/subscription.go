package subscriptions

import (
	"fmt"
	"time"
)

type SubscriptionID string

type Subscription struct {
	id     string
	signal *filterSignal
	quit   chan interface{}
	filter filter
}

type SubscriptionParams struct {
	namespace string
	filter    filter
}

func NewSubscription(namespace string, filter filter) *Subscription {
	subscriptionID := NewSubscriptionID(namespace, filter.getId())

	quit := make(chan interface{})

	return &Subscription{
		id:     subscriptionID,
		quit:   quit,
		signal: newFilterSignal(subscriptionID),
		filter: filter,
	}
}

func (s *Subscription) Start() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			filterData, err := s.filter.getChanges()
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
	return s.filter.uninstall()
}

func NewSubscriptionID(namespace, filterID string) SubscriptionID {
	return SubscriptionID(fmt.Sprintf("%s-%s", namespace, filterID))
}
