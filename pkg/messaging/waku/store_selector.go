package wakuv2

import (
	"context"
	"math/rand"
	"net"
	"runtime"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

// bootstrapDNS overrides the system resolver on mobile, where DNS resolution is
// unreliable (status-mobile #19581).
const bootstrapDNS = "8.8.8.8:53"

var overrideDNS = runtime.GOOS == "android" || runtime.GOOS == "ios"

var dnsOverrideOnce sync.Once

func applyMobileDNSOverride() {
	if !overrideDNS {
		return
	}
	dnsOverrideOnce.Do(func() {
		var dialer net.Dialer
		net.DefaultResolver = &net.Resolver{
			PreferGo: false,
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, "udp", bootstrapDNS)
			},
		}
	})
}

// storeSelector hands out storenodes to query. With no health checks there is no
// "active"/"available" node and no background cycle: it simply holds the
// configured list (set once) and offers candidates in random order so a query can
// fail over from one node to the next.
type storeSelector struct {
	mu    sync.RWMutex
	nodes []peer.AddrInfo
}

func newStoreSelector() *storeSelector {
	applyMobileDNSOverride()
	return &storeSelector{}
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

// candidates returns the storenodes to try for one query, shuffled so load is
// spread and so Query can attempt them in turn until one answers.
func (s *storeSelector) candidates() []peer.AddrInfo {
	s.mu.RLock()
	nodes := make([]peer.AddrInfo, len(s.nodes))
	copy(nodes, s.nodes)
	s.mu.RUnlock()

	rand.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
	return nodes
}
