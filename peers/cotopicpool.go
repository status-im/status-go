package peers

import (
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
)

// MailServerDiscoveryTopic topic name for mailserver discovery.
const MailServerDiscoveryTopic = "mailserver.discovery"

// MailServerDiscoveryLimits default mailserver discovery limits.
var MailServerDiscoveryLimits = params.Limits{Min: 3, Max: 3}

// newCacheOnlyTopicPool returns instance of CacheOnlyTopicPool.
func newCacheOnlyTopicPool(discovery Discovery, topic discv5.Topic, limits params.Limits, slowMode, fastMode time.Duration, cache *Cache) *CacheOnlyTopicPool {
	return &CacheOnlyTopicPool{
		TopicPool: newTopicPool(discovery, topic, limits, slowMode, fastMode, cache),
	}
}

// CacheOnlyTopicPool handles a mail server topic pool.
type CacheOnlyTopicPool struct {
	*TopicPool
}

// MaxReached checks if the max allowed peers is reached or not. When true
// peerpool will stop the discovery process on this TopicPool.
// Main difference with basic TopicPool is we want to stop discovery process
// when the number of cached peers eq/exceeds the max limit.
func (t *CacheOnlyTopicPool) MaxReached() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.limits.Max == 0 {
		return true
	}
	peers := t.cache.GetPeersRange(t.topic, t.limits.Max)
	return len(peers) >= t.limits.Max
}

var sendEnodeDiscoveryCompleted = signal.SendEnodeDiscoveredCompleted

// ConfirmAdded calls base TopicPool ConfirmAdded method and sends a signal
// confirming the enode has been discovered.
func (t *CacheOnlyTopicPool) ConfirmAdded(server *p2p.Server, nodeID discover.NodeID) {
	t.TopicPool.ConfirmAdded(server, nodeID)
	sendEnodeDiscoveryCompleted(nodeID.String(), string(t.topic))
}
