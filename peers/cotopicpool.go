package peers

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
)

// Verifier verifies if a give node is trusted.
type Verifier interface {
	VerifyNode(context.Context, discover.NodeID) bool
}

// MailServerDiscoveryTopic topic name for mailserver discovery.
const MailServerDiscoveryTopic = "whispermail"

// MailServerDiscoveryLimits default mailserver discovery limits.
var MailServerDiscoveryLimits = params.Limits{Min: 3, Max: 3}

// cacheOnlyTopicPool handles a mail server topic pool.
type cacheOnlyTopicPool struct {
	*TopicPool
	verifier Verifier
}

// newCacheOnlyTopicPool returns instance of CacheOnlyTopicPool.
func newCacheOnlyTopicPool(t *TopicPool, verifier Verifier) *cacheOnlyTopicPool {
	return &cacheOnlyTopicPool{
		TopicPool: t,
		verifier:  verifier,
	}
}

// MaxReached checks if the max allowed peers is reached or not. When true
// peerpool will stop the discovery process on this TopicPool.
// Main difference with basic TopicPool is we want to stop discovery process
// when the number of cached peers eq/exceeds the max limit.
func (t *cacheOnlyTopicPool) MaxReached() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.limits.Max == 0 {
		return true
	}
	peers := t.cache.GetPeersRange(t.topic, t.limits.Max)
	return len(peers) >= t.limits.Max
}

var sendEnodeDiscovered = signal.SendEnodeDiscovered

// ConfirmAdded calls base TopicPool ConfirmAdded method and sends a signal
// confirming the enode has been discovered.
func (t *cacheOnlyTopicPool) ConfirmAdded(server *p2p.Server, nodeID discover.NodeID) {
	trusted := t.verifier.VerifyNode(context.TODO(), nodeID)
	if trusted {
		// add to cache only if trusted
		t.TopicPool.ConfirmAdded(server, nodeID)
		sendEnodeDiscovered(nodeID.String(), string(t.topic))
	}

	id := discv5.NodeID(nodeID)
	if peer, ok := t.connectedPeers[id]; ok {
		// Delete it from `connectedPeers` immediately to
		// prevent removing it from the cache which logic is
		// implemented in TopicPool.
		delete(t.connectedPeers, id)
		t.removeServerPeer(server, peer)

		if trusted {
			t.subtractToLimits()
		}
	}
}

// subtractToLimits subtracts one to topic pool limits.
func (t *cacheOnlyTopicPool) subtractToLimits() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.limits.Max > 0 {
		t.limits.Max = t.limits.Max - 1
	}
	if t.limits.Min > 0 {
		t.limits.Min = t.limits.Min - 1
	}
}
