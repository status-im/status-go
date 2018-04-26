package peers

import (
	"sync"
	"time"
)

type syncStrategy struct {
	fastMode      time.Duration
	slowMode      time.Duration
	fastModeLimit time.Duration

	mode        chan time.Duration // for internal usage
	currentMode time.Duration      // can only be read or write in Start() goroutine
	period      chan time.Duration // deduped and returned by Start()

	mu   sync.RWMutex
	wg   sync.WaitGroup
	quit chan struct{}
}

func newSyncStrategy(fastMode, slowMode, fastModeLimit time.Duration) *syncStrategy {
	return &syncStrategy{
		fastMode:      fastMode,
		slowMode:      slowMode,
		fastModeLimit: fastModeLimit,
	}
}

// limitFastMode switches to slow mode sync after certain amount of time.
// If the timer already exists, it's canceled.
func (s *syncStrategy) limitFastMode(timeout time.Duration, cancel <-chan struct{}) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		select {
		case <-time.After(timeout):
			s.mode <- s.slowMode
		case <-cancel:
		}
	}()
}

func (s *syncStrategy) Start() <-chan time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.quit = make(chan struct{})
	// use buffered channel so it does not block on initialization
	s.period = make(chan time.Duration, 2)
	s.mode = make(chan time.Duration, 2)

	s.wg.Add(1)
	go func() {
		// This goroutine is used to sync access to `currentMode`
		// and all channels of `syncStrategy`.
		defer s.wg.Done()

		for {
			select {
			case mode := <-s.mode:
				if mode == s.currentMode {
					continue
				}

				s.period <- mode
				s.currentMode = mode

				if s.currentMode == s.fastMode {
					s.limitFastMode(s.fastModeLimit, s.quit)
				}
			case <-s.quit:
				// period must be closed as otherwise SearchTopic() form DiscV5 won't exit
				if s.period != nil {
					close(s.period)
					s.period = nil
				}
				s.currentMode = 0
				s.quit = nil
				return
			}
		}
	}()

	s.mode <- s.fastMode

	return s.period
}

func (s *syncStrategy) Stop() {
	if s.quit == nil {
		return
	}
	close(s.quit)
	s.wg.Wait()
}

// Update switches between modes depending on number of connected peers and limits.
// If `connectedPeers` is lower than min, it chooses a fast mode.
func (s *syncStrategy) Update(connectedPeers, min, max int) {
	newMode := s.slowMode
	if connectedPeers < min {
		newMode = s.fastMode
	}
	s.mode <- newMode
}
