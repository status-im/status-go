package peers

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/geth/params"
)

// NewTopicPool returns instance of TopicPool
func NewTopicPool(topic discv5.Topic, limits params.Limits, slowMode, fastMode time.Duration) *TopicPool {
	pool := TopicPool{
		topic:           topic,
		limits:          limits,
		fastMode:        fastMode,
		slowMode:        slowMode,
		fastModeTimeout: DefaultTopicFastModeTimeout,
		peerPool:        make(map[discv5.NodeID]*peerInfoItem),
		peerPoolQueue:   make(peerPriorityQueue, 0),
		connectedPeers:  make(map[discv5.NodeID]*peerInfo),
	}

	heap.Init(&pool.peerPoolQueue)

	return &pool
}

// TopicPool manages peers for topic.
type TopicPool struct {
	// configuration
	topic           discv5.Topic
	limits          params.Limits
	fastMode        time.Duration
	slowMode        time.Duration
	fastModeTimeout time.Duration

	mu   sync.RWMutex
	wg   sync.WaitGroup // wait group for all goroutines spawn during TopicPool operation
	quit chan struct{}

	running int32

	currentMode           time.Duration
	period                chan time.Duration
	fastModeTimeoutCancel chan struct{}

	peerPool       map[discv5.NodeID]*peerInfoItem // found but not connected peers
	peerPoolQueue  peerPriorityQueue               // priority queue to find the most recent peer
	connectedPeers map[discv5.NodeID]*peerInfo     // currently connected peers

	cache *Cache
}

func (t *TopicPool) addToPeerPool(peer *peerInfo) {
	if _, ok := t.peerPool[peer.node.ID]; ok {
		return
	}
	item := &peerInfoItem{peerInfo: peer}
	t.peerPool[peer.node.ID] = item
	heap.Push(&t.peerPoolQueue, item)
}

func (t *TopicPool) removeFromPeerPool(nodeID discv5.NodeID) {
	peer, ok := t.peerPool[nodeID]
	if !ok {
		return
	}
	delete(t.peerPool, nodeID)
	heap.Remove(&t.peerPoolQueue, peer.index)
}

func (t *TopicPool) popFromPeerPool() *peerInfo {
	if t.peerPoolQueue.Len() == 0 {
		return nil
	}
	item := heap.Pop(&t.peerPoolQueue).(*peerInfoItem)
	delete(t.peerPool, item.node.ID)
	return item.peerInfo
}

func (t *TopicPool) updatePeerInPool(nodeID discv5.NodeID, time mclock.AbsTime) {
	peer, ok := t.peerPool[nodeID]
	if !ok {
		return
	}
	peer.discoveredTime = mclock.Now()
	heap.Fix(&t.peerPoolQueue, peer.index)
}

func (t *TopicPool) movePeerFromPoolToConnected(nodeID discv5.NodeID) {
	peer, ok := t.peerPool[nodeID]
	if !ok {
		return
	}
	t.removeFromPeerPool(nodeID)
	t.connectedPeers[nodeID] = peer.peerInfo
}

// SearchRunning returns true if search is running
func (t *TopicPool) SearchRunning() bool {
	return atomic.LoadInt32(&t.running) == 1
}

// MaxReached returns true if we connected with max number of peers.
func (t *TopicPool) MaxReached() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.connectedPeers) == t.limits.Max
}

// BelowMin returns true if current number of peers is below min limit.
func (t *TopicPool) BelowMin() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.connectedPeers) < t.limits.Min
}

// updateSyncMode changes the sync mode depending on the current number
// of connected peers and limits.
func (t *TopicPool) updateSyncMode() {
	newMode := t.slowMode
	if len(t.connectedPeers) < t.limits.Min {
		newMode = t.fastMode
	}

	t.setSyncMode(newMode)
}

func (t *TopicPool) setSyncMode(mode time.Duration) {
	if mode == t.currentMode {
		return
	}

	t.period <- mode
	t.currentMode = mode

	if mode == t.fastMode {
		t.limitFastMode(t.fastModeTimeout)
	}
}

func (t *TopicPool) limitFastMode(timeout time.Duration) {
	if timeout == 0 {
		return
	}

	if t.fastModeTimeoutCancel != nil {
		close(t.fastModeTimeoutCancel)
	}
	cancel := make(chan struct{})
	t.fastModeTimeoutCancel = cancel

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		select {
		case <-time.After(timeout):
			t.mu.Lock()
			t.setSyncMode(t.slowMode)
			t.mu.Unlock()
		case <-cancel:
			return
		}
	}()
}

// ConfirmAdded called when peer was added by p2p Server.
// 1. Skip a peer if it not in our peer table
// 2. Add a peer to a cache.
// 3. Disconnect a peer if it was connected after we reached max limit of peers.
//    (we can't know in advance if peer will be connected, thats why we allow
//     to overflow for short duration)
// 4. Switch search to slow mode if it is running.
func (t *TopicPool) ConfirmAdded(server *p2p.Server, nodeID discover.NodeID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	discV5NodeID := discv5.NodeID(nodeID)

	// inbound connection
	peer, exist := t.peerPool[discV5NodeID]
	if !exist {
		return
	}

	// established connection means that the node
	// is a viable candidate for a connection and can be cached
	if t.cache != nil {
		if err := t.cache.AddPeer(peer.node, t.topic); err != nil {
			log.Error("failed to persist a peer", "error", err)
		}
	}

	// if the upper limit is already reached, drop this peer
	if len(t.connectedPeers) == t.limits.Max {
		log.Debug("max limit is reached drop the peer", "ID", nodeID, "topic", t.topic)
		peer.dismissed = true
		t.removeServerPeer(server, peer.peerInfo)
		return
	}

	t.movePeerFromPoolToConnected(discV5NodeID)
	// make sure `dismissed` is reset
	peer.dismissed = false

	// A peer was added so check if we can switch to slow mode.
	if t.SearchRunning() {
		t.updateSyncMode()
	}
}

// ConfirmDropped called when server receives drop event.
// 1. Skip peer if it is not in our peer table.
// 2. If disconnect request - we could drop that peer ourselves.
// 3. If connected number will drop below min limit - switch to fast mode.
// 4. Delete a peer from cache and peer table.
// Returns false if peer is not in our table or we requested removal of this peer.
// Otherwise peer is removed and true is returned.
func (t *TopicPool) ConfirmDropped(server *p2p.Server, nodeID discover.NodeID) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	discV5NodeID := discv5.NodeID(nodeID)

	// either inbound or connected from another topic
	peer, exist := t.connectedPeers[discV5NodeID]
	if !exist {
		return false
	}

	log.Debug("disconnect", "ID", nodeID, "dismissed", peer.dismissed)

	// Peer was removed by us because exceeded the limit.
	// Add it back to the pool as it can be useful in the future.
	if peer.dismissed {
		t.addToPeerPool(peer)
		return false
	}

	// If there was a network error, this event will be received
	// but the peer won't be removed from the static nodes set.
	// That's why we need to call `removeServerPeer` manually.
	t.removeServerPeer(server, peer)

	delete(t.connectedPeers, discV5NodeID)
	if t.cache != nil {
		if err := t.cache.RemovePeer(discV5NodeID, t.topic); err != nil {
			log.Error("failed to remove peer from cache", "error", err)
		}
	}

	// As we removed a peer, update a sync strategy if needed.
	if t.SearchRunning() {
		t.updateSyncMode()
	}

	return true
}

// AddPeerFromTable checks if there is a valid peer in local table and adds it to a server.
func (t *TopicPool) AddPeerFromTable(server *p2p.Server) *discv5.Node {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// The most recently added peer is removed from the queue.
	// If it did not expire yet, it will be added to the server.
	// TODO(adam): investigate if it's worth to keep the peer in the queue
	// until the server confirms it is added and in the meanwhile only adjust its priority.
	peer := t.popFromPeerPool()
	if peer != nil && mclock.Now() < peer.discoveredTime+mclock.AbsTime(expirationPeriod) {
		t.addServerPeer(server, peer)
		return peer.node
	}

	return nil
}

// StartSearch creates discv5 queries and runs a loop to consume found peers.
func (t *TopicPool) StartSearch(server *p2p.Server) error {
	if atomic.LoadInt32(&t.running) == 1 {
		return nil
	}
	if server.DiscV5 == nil {
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

	if t.cache != nil {
		for _, peer := range t.cache.GetPeersRange(t.topic, 5) {
			log.Debug("adding a peer from cache", "peer", peer)
			found <- peer
		}
	}

	t.wg.Add(1)
	go func() {
		server.DiscV5.SearchTopic(t.topic, t.period, found, lookup)
		t.wg.Done()
	}()
	t.wg.Add(1)
	go func() {
		t.handleFoundPeers(server, found, lookup)
		t.wg.Done()
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

// processFoundNode called when node is discovered by kademlia search query
// 2 important conditions
// 1. every time when node is processed we need to update discoveredTime.
//    peer will be considered as valid later only if it was discovered < 60m ago
// 2. if peer is connected or if max limit is reached we are not a adding peer to p2p server
func (t *TopicPool) processFoundNode(server *p2p.Server, node *discv5.Node) {
	t.mu.Lock()
	defer t.mu.Unlock()

	log.Debug("peer found", "ID", node.ID, "topic", t.topic)

	// peer is already connected so update only discoveredTime
	if peer, ok := t.connectedPeers[node.ID]; ok {
		peer.discoveredTime = mclock.Now()
		return
	}

	if _, ok := t.peerPool[node.ID]; ok {
		t.updatePeerInPool(node.ID, mclock.Now())
	} else {
		t.addToPeerPool(&peerInfo{
			discoveredTime: mclock.Now(),
			node:           node,
		})
	}

	// the upper limit is not reached, so let's add this peer
	if len(t.connectedPeers) < t.limits.Max {
		t.addServerPeer(server, t.peerPool[node.ID].peerInfo)
	}
}

func (t *TopicPool) addServerPeer(server *p2p.Server, info *peerInfo) {
	server.AddPeer(discover.NewNode(
		discover.NodeID(info.node.ID),
		info.node.IP,
		info.node.UDP,
		info.node.TCP,
	))
}

func (t *TopicPool) removeServerPeer(server *p2p.Server, info *peerInfo) {
	server.RemovePeer(discover.NewNode(
		discover.NodeID(info.node.ID),
		info.node.IP,
		info.node.UDP,
		info.node.TCP,
	))
}

// StopSearch stops the closes stop
func (t *TopicPool) StopSearch() {
	if !atomic.CompareAndSwapInt32(&t.running, 1, 0) {
		return
	}
	if t.quit == nil {
		return
	}
	select {
	case <-t.quit:
		return
	default:
		log.Debug("stoping search", "topic", t.topic)
		if t.fastModeTimeoutCancel != nil {
			close(t.fastModeTimeoutCancel)
			t.fastModeTimeoutCancel = nil
		}
		close(t.period)
		t.period = nil
		t.currentMode = 0
		close(t.quit)
	}
	t.wg.Wait()
}
