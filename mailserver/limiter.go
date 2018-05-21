package mailserver

import (
	"sync"
	"time"
)

type limiter struct {
	mu sync.RWMutex

	timeout time.Duration
	db      map[string]time.Time
}

func newLimiter(timeout time.Duration) *limiter {
	return &limiter{
		timeout: timeout,
		db:      make(map[string]time.Time),
	}
}

func (l *limiter) add(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.db[id] = time.Now()
}

func (l *limiter) isAllowed(id string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if lastRequestTime, ok := l.db[id]; ok {
		return lastRequestTime.Add(l.timeout).Before(time.Now())
	}

	return true
}

func (l *limiter) deleteExpired() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for id, lastRequestTime := range l.db {
		if lastRequestTime.Add(l.timeout).Before(now) {
			delete(l.db, id)
		}
	}
}
