package peers

import (
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
)

// ProxyTopicPool acts like a proxy between two Discovery V5 instances.
// Every found node is added immediately to another Discovery V5 instance.
type ProxyTopicPool struct {
	*TopicPool
	destDiscovery Discovery
	foundNodes    int
}

// NewProxyTopicPool creates a new ProxyTopicPool.
func NewProxyTopicPool(t *TopicPool, d Discovery) *ProxyTopicPool {
	return &ProxyTopicPool{
		TopicPool:     t,
		destDiscovery: d,
	}
}

// StartSearch creates discv5 queries and runs a loop to consume found peers.
func (t *ProxyTopicPool) StartSearch(server *p2p.Server) error {
	if atomic.LoadInt32(&t.running) == 1 {
		return nil
	}
	if !t.discovery.Running() {
		return ErrDiscv5NotRunning
	}
	atomic.StoreInt32(&t.running, 1)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.quit = make(chan struct{})

	// `period` is used to notify about the current sync mode.
	t.period = make(chan time.Duration, 2)
	// use fast sync mode at the beginning
	t.setSyncMode(t.fastMode)

	// peers management
	found := make(chan *discv5.Node, 5) // 5 reasonable number for concurrently found nodes
	lookup := make(chan bool, 10)       // sufficiently buffered channel, just prevents blocking because of lookup

	t.discWG.Add(1)
	go func() {
		if err := t.discovery.Discover(string(t.topic), t.period, found, lookup); err != nil {
			log.Error("error searching foro", "topic", t.topic, "err", err)
		}
		t.discWG.Done()
	}()
	t.poolWG.Add(1)
	go func() {
		t.handleFoundPeers(server, found, lookup)
		t.poolWG.Done()
	}()

	return nil
}

func (t *TopicPool) handleFoundPeers(server *p2p.Server, found <-chan *discv5.Node, lookup <-chan bool) {
	selfID := discv5.NodeID(server.Self().ID)
	for {
		select {
		case <-t.quit:
			return
		case <-lookup:
		case node := <-found:
			if node.ID != selfID {
				t.processFoundNode(server, node)
			}
		}
	}
}

func (t *ProxyTopicPool) processFoundNode(server *p2p.Server, node *discv5.Node) {
	t.mu.Lock()
	log.Debug("ProxyTopicPool passes node", "node", node, "found", t.foundNodes)
	t.foundNodes++
	if t.destDiscovery != nil {
		log.Debug("ProxyTopicPool inserts node", "topic", t.topic)
		t.destDiscovery.InsertNodes(t.topic, []*discv5.Node{node})
	}
	t.mu.Unlock()
}

// ConfirmAdded overrides default TopicPool behavior and acts like noop.
func (t *ProxyTopicPool) ConfirmAdded(server *p2p.Server, nodeID discover.NodeID) {
	return
}

// ConfirmDropped overrides default TopicPool behavior and acts like noop.
func (t *ProxyTopicPool) ConfirmDropped(server *p2p.Server, nodeID discover.NodeID) bool {
	return false
}

// ConnectedPeers overrides default TopicPool behavior and acts like noop.
func (t *ProxyTopicPool) ConnectedPeers() []*discv5.Node {
	return nil
}

// AddPeerFromTable overrides default TopicPool behavior and acts like noop.
func (t *ProxyTopicPool) AddPeerFromTable(server *p2p.Server) *discv5.Node {
	return nil
}

// BelowMin returns true if number of found nodes is below lower limit.
func (t *ProxyTopicPool) BelowMin() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.foundNodes < t.limits.Min
}

// MaxReached always returns false as NewProxyTopicPool
// only discovers and passes found peers along.
func (t *ProxyTopicPool) MaxReached() bool {
	return false
}
