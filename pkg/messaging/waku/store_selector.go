package wakuv2

import (
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// storeFailureBackoff is how long a storenode is deprioritized after a failed query.
const storeFailureBackoff = 2 * time.Minute

// storeSelector hands out storenodes to query. With no health checks there is no
// "active"/"available" node and no background cycle: it holds the configured list
// (set once, shuffled so different clients prefer different nodes) and offers
// candidates in that fixed order, deprioritizing nodes that recently failed a
// query so a healthy node is tried first.
type storeSelector struct {
	mu           sync.RWMutex
	nodes        []peer.AddrInfo
	backoffUntil map[peer.ID]time.Time
}

func newStoreSelector() *storeSelector {
	return &storeSelector{backoffUntil: make(map[peer.ID]time.Time)}
}

// setStorenodes replaces the candidate list. Called once at startup. The list is
// copied (so later caller mutation can't change it) and shuffled once, so
// different clients prefer different storenodes — load spreads across the fleet
// without the per-query churn of reshuffling on every call.
func (s *storeSelector) setStorenodes(nodes []peer.AddrInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes = slices.Clone(nodes)
	rand.Shuffle(len(s.nodes), func(i, j int) { s.nodes[i], s.nodes[j] = s.nodes[j], s.nodes[i] })
}

func (s *storeSelector) hasStorenodes() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.nodes) > 0
}

// markFailure deprioritizes a storenode for storeFailureBackoff after a failed query.
func (s *storeSelector) markFailure(id peer.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backoffUntil[id] = time.Now().Add(storeFailureBackoff)
}

// markSuccess clears any backoff for a storenode after a successful query.
func (s *storeSelector) markSuccess(id peer.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.backoffUntil, id)
}

// candidates returns the storenodes to try for one query, in the fixed startup
// order: nodes not in failure backoff first, then recently-failed nodes as a
// last resort. Query sticks to the first healthy node and only moves on when it
// fails (which backs it off, demoting it here) or once its backoff expires.
func (s *storeSelector) candidates() []peer.AddrInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	var ready, backedOff []peer.AddrInfo
	for _, n := range s.nodes {
		if until, ok := s.backoffUntil[n.ID]; ok && now.Before(until) {
			backedOff = append(backedOff, n)
		} else {
			ready = append(ready, n)
		}
	}
	return append(ready, backedOff...)
}
