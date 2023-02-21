package timecache

import (
	"sync"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
)

// LastSeenCache is a LRU cache that keeps entries for up to a specified time duration. After this duration has elapsed,
// "old" entries will be purged from the cache.
//
// It's also a "sliding window" cache. Every time an unexpired entry is seen again, its timestamp slides forward. This
// keeps frequently occurring entries cached and prevents them from being propagated, especially because of network
// issues that might increase the number of duplicate messages in the network.
//
// Garbage collection of expired entries is event-driven, i.e. it only happens when there is a new entry added to the
// cache. This should be ok - if existing entries are being looked up then the cache is not growing, and when a new one
// appears that would grow the cache, garbage collection will attempt to reduce the pressure on the cache.
//
// This implementation is heavily inspired by https://github.com/whyrusleeping/timecache.
type LastSeenCache struct {
	m     *linkedhashmap.Map
	span  time.Duration
	guard *sync.Mutex
}

func newLastSeenCache(span time.Duration) TimeCache {
	return &LastSeenCache{
		m:     linkedhashmap.New(),
		span:  span,
		guard: new(sync.Mutex),
	}
}

func (tc *LastSeenCache) Add(s string) {
	tc.guard.Lock()
	defer tc.guard.Unlock()

	tc.add(s)

	// Garbage collect expired entries
	// TODO(#515): Do GC in the background
	tc.gc()
}

func (tc *LastSeenCache) add(s string) {
	// We don't need a lock here because this function is always called with the lock already acquired.

	// If an entry already exists, remove it and add a new one to the back of the list to maintain temporal ordering and
	// an accurate sliding window.
	tc.m.Remove(s)
	now := time.Now()
	tc.m.Put(s, &now)
}

func (tc *LastSeenCache) gc() {
	// We don't need a lock here because this function is always called with the lock already acquired.
	iter := tc.m.Iterator()
	for iter.Next() {
		key := iter.Key()
		ts := iter.Value().(*time.Time)
		// Exit if we've found an entry with an unexpired timestamp. Since we're iterating in order of insertion, all
		// entries hereafter will be unexpired.
		if time.Since(*ts) <= tc.span {
			return
		}
		tc.m.Remove(key)
	}
}

func (tc *LastSeenCache) Has(s string) bool {
	tc.guard.Lock()
	defer tc.guard.Unlock()

	// If the entry exists and has not already expired, slide it forward.
	if ts, found := tc.m.Get(s); found {
		if t := ts.(*time.Time); time.Since(*t) <= tc.span {
			tc.add(s)
			return true
		}
	}
	return false
}
