package protocol

import (
	"sync"
)

type Subscription struct {
	sync.RWMutex

	err  error
	done chan struct{}
}

func NewSubscription() *Subscription {
	return &Subscription{
		done: make(chan struct{}),
	}
}

func (s *Subscription) Cancel(err error) {
	s.Lock()
	defer s.Unlock()

	if s.done == nil {
		return
	}

	close(s.done)
	s.done = nil
	s.err = err
}

func (s *Subscription) Unsubscribe() {
	s.Lock()
	defer s.Unlock()

	if s.done == nil {
		return
	}

	close(s.done)
	s.done = nil
}

func (s *Subscription) Err() error {
	s.RLock()
	defer s.RUnlock()
	return s.err
}

func (s *Subscription) Done() <-chan struct{} {
	s.RLock()
	defer s.RUnlock()
	return s.done
}
