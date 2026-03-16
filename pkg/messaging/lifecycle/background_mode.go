package lifecycle

import (
	"sync"
	"sync/atomic"
)

var pausedBackground atomic.Bool
var subscriberID atomic.Uint64

var subscribersMu sync.RWMutex
var subscribers = map[uint64]chan bool{}

type PausedBackgroundSubscription interface {
	C() <-chan bool
	Unsubscribe()
}

type pausedBackgroundSubscription struct {
	id   uint64
	ch   chan bool
	once sync.Once
}

func SetPausedBackground(paused bool) {
	if pausedBackground.Swap(paused) == paused {
		return
	}

	subscribersMu.RLock()
	defer subscribersMu.RUnlock()

	for _, ch := range subscribers {
		sendLatestPausedState(ch, paused)
	}
}

func IsPausedBackground() bool {
	return pausedBackground.Load()
}

func SubscribePausedBackground() PausedBackgroundSubscription {
	id := subscriberID.Add(1)
	ch := make(chan bool, 1)

	subscribersMu.Lock()
	subscribers[id] = ch
	subscribersMu.Unlock()

	sendLatestPausedState(ch, IsPausedBackground())

	return &pausedBackgroundSubscription{
		id: id,
		ch: ch,
	}
}

func (s *pausedBackgroundSubscription) C() <-chan bool {
	return s.ch
}

func (s *pausedBackgroundSubscription) Unsubscribe() {
	s.once.Do(func() {
		subscribersMu.Lock()
		ch, ok := subscribers[s.id]
		if ok {
			delete(subscribers, s.id)
		}
		subscribersMu.Unlock()
		if ok {
			close(ch)
		}
	})
}

func sendLatestPausedState(ch chan bool, paused bool) {
	select {
	case ch <- paused:
		return
	default:
	}

	// Coalesce pending value for slow subscribers.
	select {
	case <-ch:
	default:
	}

	select {
	case ch <- paused:
	default:
	}
}
