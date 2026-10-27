package messaging

import (
	"sync"
	"time"
)

const (
	requestedHashesTTL     = 30 * time.Minute
	requestedHashesMaxSize = 10_000
)

type requestedHashEntry struct {
	hash        string
	requestedAt time.Time
}

// requestedHashes remembers envelope hashes recently requested from storenodes
// so missing SDS dependencies are fetched at most once per TTL, regardless of
// the outcome. This prevents re-fetching envelopes that fail processing (and so
// never reach the processed-envelopes cache) or that storenodes do not have.
type requestedHashes struct {
	mu      sync.Mutex
	ttl     time.Duration
	maxSize int
	now     func() time.Time
	entries map[string]time.Time
	// order holds entries in insertion order. Since the TTL is uniform this is
	// also expiry order. It may contain stale items for re-claimed hashes;
	// those are detected by comparing against entries.
	order []requestedHashEntry
}

func newRequestedHashes(ttl time.Duration, maxSize int) *requestedHashes {
	return &requestedHashes{
		ttl:     ttl,
		maxSize: maxSize,
		now:     time.Now,
		entries: make(map[string]time.Time),
	}
}

// claim returns the subset of hashes that were not requested within the TTL
// and marks them as requested. Concurrent callers never get the same hash.
func (r *requestedHashes) claim(hashes []string) []string {
	if r == nil {
		return hashes
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	r.prune(now)

	claimed := make([]string, 0, len(hashes))
	for _, h := range hashes {
		if requestedAt, ok := r.entries[h]; ok && now.Sub(requestedAt) < r.ttl {
			continue
		}
		r.entries[h] = now
		r.order = append(r.order, requestedHashEntry{hash: h, requestedAt: now})
		claimed = append(claimed, h)
	}

	r.prune(now)
	return claimed
}

// prune drops expired entries and, if over capacity, the oldest ones.
// Caller must hold r.mu.
func (r *requestedHashes) prune(now time.Time) {
	i := 0
	for ; i < len(r.order); i++ {
		e := r.order[i]
		current, ok := r.entries[e.hash]
		if !ok || !current.Equal(e.requestedAt) {
			continue // stale queue item, hash was re-claimed or already removed
		}
		if now.Sub(e.requestedAt) < r.ttl && len(r.entries) <= r.maxSize {
			break
		}
		delete(r.entries, e.hash)
	}
	if i > 0 {
		r.order = append(r.order[:0:0], r.order[i:]...)
	}
}

// release forgets hashes so they can be claimed again right away.
func (r *requestedHashes) release(hashes []string) {
	if r == nil || len(hashes) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, h := range hashes {
		delete(r.entries, h)
	}
}

// expireIn makes hashes claimable again after d instead of after the full TTL.
func (r *requestedHashes) expireIn(hashes []string, d time.Duration) {
	if r == nil || len(hashes) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	requestedAt := r.now().Add(d - r.ttl)
	for _, h := range hashes {
		if _, ok := r.entries[h]; !ok {
			continue
		}
		r.entries[h] = requestedAt
		// Out of expiry order, so prune may drop it late; claim checks the TTL itself.
		r.order = append(r.order, requestedHashEntry{hash: h, requestedAt: requestedAt})
	}
}

func (r *requestedHashes) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}
