package subscriptions

import (
	"errors"
	"fmt"
	"time"
)

type SubscriptionID string

type Subscription struct {
	id      SubscriptionID
	signal  *filterSignal
	quit    chan struct{}
	filter  filter
	stopped bool
}

func NewSubscription(namespace string, filter filter) *Subscription {
	subscriptionID := NewSubscriptionID(namespace, filter.getID())

	quit := make(chan struct{})

	return &Subscription{
		id:     subscriptionID,
		quit:   quit,
		signal: newFilterSignal(string(subscriptionID)),
		filter: filter,
	}
}

func (s *Subscription) Start(checkPeriod time.Duration) error {
	if s.stopped {
		return errors.New("it is impossible to start an already stopped subscription")
	}
	ticker := time.NewTicker(checkPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			filterData, err := s.filter.getChanges()
			if err != nil {
				s.signal.SendError(err)
			} else if len(filterData) > 0 {
				s.signal.SendData(filterData)
			}
		case <-s.quit:
			return nil
		}
	}
}

func (s *Subscription) Stop(uninstall bool) error {
	if s.stopped {
		return nil
	}
	close(s.quit)
	if uninstall {
		return s.filter.uninstall()
	}
	s.stopped = true
	return nil
}

func NewSubscriptionID(namespace, filterID string) SubscriptionID {
	return SubscriptionID(fmt.Sprintf("%s-%s", namespace, filterID))
}
