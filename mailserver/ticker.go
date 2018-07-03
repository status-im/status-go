package mailserver

import (
	"sync"
	"time"
)

type ticker struct {
	mu         sync.RWMutex
	timeTicker *time.Ticker
}

func (t *ticker) run(period time.Duration, fn func()) {
	if t.timeTicker != nil {
		return
	}

	tt := time.NewTicker(period)
	t.mu.Lock()
	t.timeTicker = tt
	t.mu.Unlock()
	go func() {
		for range tt.C {
			fn()
		}
	}()
}

func (t *ticker) stop() {
	t.mu.RLock()
	t.timeTicker.Stop()
	t.mu.RUnlock()
}
