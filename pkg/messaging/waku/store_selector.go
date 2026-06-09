package wakuv2

import (
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// storeFailureBackoff is how long a storenode is deprioritized after a failed query.
const storeFailureBackoff = 2 * time.Minute

// storeSelector hands out storenodes to query. With no health checks there is no
// "active"/"available" node and no background cycle: it holds the configured list
// (set once) and offers candidates in random order, deprioritizing nodes that
// recently failed a query so a healthy node is tried first.
type storeSelector struct {
	mu           sync.RWMutex
	nodes        []peer.AddrInfo
	backoffUntil map[peer.ID]time.Time
}

func newStoreSelector() *storeSelector {
	return &storeSelector{backoffUntil: make(map[peer.ID]time.Time)}
}

// setStorenodes replaces the candidate list. Called once at startup.
func (s *storeSelector) setStorenodes(nodes []peer.AddrInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes = nodes
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

// candidates returns the storenodes to try for one query: nodes not in failure
// backoff first (shuffled to spread load), then recently-failed nodes (also
// shuffled) as a last resort — so Query prefers healthy nodes but can still fall
// back when every node has failed recently.
func (s *storeSelector) candidates() []peer.AddrInfo {
	s.mu.RLock()
	now := time.Now()
	var ready, backedOff []peer.AddrInfo
	for _, n := range s.nodes {
		if until, ok := s.backoffUntil[n.ID]; ok && now.Before(until) {
			backedOff = append(backedOff, n)
		} else {
			ready = append(ready, n)
		}
	}
	s.mu.RUnlock()

	rand.Shuffle(len(ready), func(i, j int) { ready[i], ready[j] = ready[j], ready[i] })
	rand.Shuffle(len(backedOff), func(i, j int) { backedOff[i], backedOff[j] = backedOff[j], backedOff[i] })
	return append(ready, backedOff...)
}
