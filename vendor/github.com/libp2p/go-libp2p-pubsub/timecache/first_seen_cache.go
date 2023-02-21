package timecache

import (
	"container/list"
	"sync"
	"time"
)

// FirstSeenCache is a thread-safe copy of https://github.com/whyrusleeping/timecache.
type FirstSeenCache struct {
	q     *list.List
	m     map[string]time.Time
	span  time.Duration
	guard *sync.RWMutex
}

func newFirstSeenCache(span time.Duration) TimeCache {
	return &FirstSeenCache{
		q:     list.New(),
		m:     make(map[string]time.Time),
		span:  span,
		guard: new(sync.RWMutex),
	}
}

func (tc FirstSeenCache) Add(s string) {
	tc.guard.Lock()
	defer tc.guard.Unlock()

	_, ok := tc.m[s]
	if ok {
		panic("putting the same entry twice not supported")
	}

	// TODO(#515): Do GC in the background
	tc.sweep()

	tc.m[s] = time.Now()
	tc.q.PushFront(s)
}

func (tc FirstSeenCache) sweep() {
	for {
		back := tc.q.Back()
		if back == nil {
			return
		}

		v := back.Value.(string)
		t, ok := tc.m[v]
		if !ok {
			panic("inconsistent cache state")
		}

		if time.Since(t) > tc.span {
			tc.q.Remove(back)
			delete(tc.m, v)
		} else {
			return
		}
	}
}

func (tc FirstSeenCache) Has(s string) bool {
	tc.guard.RLock()
	defer tc.guard.RUnlock()

	ts, ok := tc.m[s]
	// Only consider the entry found if it was present in the cache AND hadn't already expired.
	return ok && time.Since(ts) <= tc.span
}
