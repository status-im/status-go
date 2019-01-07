package mailserver

import (
	"sync"
	"time"
)

type rateLimiter struct {
	sync.RWMutex

	duration time.Duration // duration of the limit
	db       map[string]time.Time

	cancel chan struct{}
}

func newRateLimiter(duration time.Duration) *rateLimiter {
	return &rateLimiter{
		duration: duration,
		db:       make(map[string]time.Time),
	}
}

func (l *rateLimiter) Start() {
	l.Lock()
	l.cancel = make(chan struct{})
	l.Unlock()
	go l.cleanUp(time.Second, l.cancel)
}

func (l *rateLimiter) Stop() {
	l.Lock()
	defer l.Unlock()

	if l.cancel == nil {
		return
	}
	close(l.cancel)
	l.cancel = nil
}

func (l *rateLimiter) Add(id string) {
	l.Lock()
	l.db[id] = time.Now()
	l.Unlock()
}

func (l *rateLimiter) IsAllowed(id string) bool {
	l.RLock()
	defer l.RUnlock()

	if lastRequestTime, ok := l.db[id]; ok {
		return lastRequestTime.Add(l.duration).Before(time.Now())
	}

	return true
}

func (l *rateLimiter) cleanUp(period time.Duration, cancel <-chan struct{}) {
	t := time.NewTicker(period)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			l.deleteExpired()
		case <-cancel:
			return
		}
	}
}

func (l *rateLimiter) deleteExpired() {
	l.Lock()
	defer l.Unlock()

	now := time.Now()
	for id, lastRequestTime := range l.db {
		if lastRequestTime.Add(l.duration).Before(now) {
			delete(l.db, id)
		}
	}
}
