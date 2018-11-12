package peers

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"

	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/discovery"
	"github.com/status-im/status-go/params"
)

const (
	// notQueuedIndex used to define that item is not queued in the heap queue.
	notQueuedIndex = -1
)

// maxCachedPeersMultiplier peers max limit will be multiplied by this number
// to get the maximum number of cached peers allowed.
var maxCachedPeersMultiplier = 2

// TopicPoolInterface the TopicPool interface.
type TopicPoolInterface interface {
	StopSearch(server *p2p.Server)
	BelowMin() bool
	SearchRunning() bool
	StartSearch(server *p2p.Server) error
	ConfirmDropped(server *p2p.Server, nodeID enode.ID) bool
	AddPeerFromTable(server *p2p.Server) *discv5.Node
	MaxReached() bool
	ConfirmAdded(server *p2p.Server, nodeID enode.ID)
	isStopped() bool
	Topic() discv5.Topic
	SetLimits(limits params.Limits)
	setStopSearchTimeout(delay time.Duration)
	readyToStopSearch() bool
}

// newTopicPool returns instance of TopicPool.
func newTopicPool(discovery discovery.Discovery, topic discv5.Topic, limits params.Limits, slowMode, fastMode time.Duration, cache *Cache) *TopicPool {
	pool := TopicPool{
		discovery:            discovery,
		topic:                topic,
		limits:               limits,
		fastMode:             fastMode,
		slowMode:             slowMode,
		fastModeTimeout:      DefaultTopicFastModeTimeout,
		pendingPeers:         make(map[enode.ID]*peerInfoItem),
		discoveredPeersQueue: make(peerPriorityQueue, 0),
		connectedPeers:       make(map[enode.ID]*peerInfo),
		cache:                cache,
		maxCachedPeers:       limits.Max * maxCachedPeersMultiplier,
	}
	heap.Init(&pool.discoveredPeersQueue)

	return &pool
}

// TopicPool manages peers for topic.
type TopicPool struct {
	discovery discovery.Discovery

	// configuration
	topic           discv5.Topic
	limits          params.Limits
	fastMode        time.Duration
	slowMode        time.Duration
	fastModeTimeout time.Duration

	mu     sync.RWMutex
	discWG sync.WaitGroup
	poolWG sync.WaitGroup
	quit   chan struct{}

	running int32

	currentMode           time.Duration
	period                chan time.Duration
	fastModeTimeoutCancel chan struct{}

	pendingPeers         map[enode.ID]*peerInfoItem // contains found and requested to be connected peers but not confirmed
	discoveredPeersQueue peerPriorityQueue          // priority queue to find the most recently discovered peers; does not containt peers requested to connect
	connectedPeers       map[enode.ID]*peerInfo     // currently connected peers

	stopSearchTimeout *time.Time

	maxCachedPeers int
	cache          *Cache
}

func (t *TopicPool) addToPendingPeers(peer *peerInfo) {
	en, err := Discv5ToEnode(*peer.node)
	if err != nil {
		log.Warn("Failed to decode key", "ID", peer.node.ID, "err", err)
		return
	}

	if _, ok := t.pendingPeers[en.ID()]; ok {
		return
	}
	t.pendingPeers[en.ID()] = &peerInfoItem{
		peerInfo: peer,
		index:    notQueuedIndex,
	}
}

// addToQueue adds the passed peer to the queue if it is already pending.
func (t *TopicPool) addToQueue(peer *peerInfo) {
	en, err := Discv5ToEnode(*peer.node)
	if err != nil {
		log.Warn("Failed to decode key", "ID", peer.node.ID, "err", err)
		return
	}

	if p, ok := t.pendingPeers[en.ID()]; ok {
		heap.Push(&t.discoveredPeersQueue, p)
	}
}

func (t *TopicPool) popFromQueue() *peerInfo {
	if t.discoveredPeersQueue.Len() == 0 {
		return nil
	}
	item := heap.Pop(&t.discoveredPeersQueue).(*peerInfoItem)
	item.index = notQueuedIndex
	return item.peerInfo
}

func (t *TopicPool) removeFromPendingPeers(nodeID enode.ID) {
	peer, ok := t.pendingPeers[nodeID]
	if !ok {
		return
	}
	delete(t.pendingPeers, nodeID)
	if peer.index != notQueuedIndex {
		heap.Remove(&t.discoveredPeersQueue, peer.index)
	}
}

func (t *TopicPool) updatePendingPeer(nodeID enode.ID, time mclock.AbsTime) {
	peer, ok := t.pendingPeers[nodeID]
	if !ok {
		return
	}
	peer.discoveredTime = mclock.Now()
	if peer.index != notQueuedIndex {
		heap.Fix(&t.discoveredPeersQueue, peer.index)
	}
}

func (t *TopicPool) movePeerFromPoolToConnected(nodeID enode.ID) {
	peer, ok := t.pendingPeers[nodeID]
	if !ok {
		return
	}
	t.removeFromPendingPeers(nodeID)
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

// maxCachedPeersReached returns true if max number of cached peers is reached.
func (t *TopicPool) maxCachedPeersReached() bool {
	if t.maxCachedPeers == 0 {
		return true
	}
	peers := t.cache.GetPeersRange(t.topic, t.maxCachedPeers)

	return len(peers) >= t.maxCachedPeers
}

// setStopSearchTimeout sets the timeout to stop current topic search if it's not
// been stopped before.
func (t *TopicPool) setStopSearchTimeout(delay time.Duration) {
	if t.stopSearchTimeout != nil {
		return
	}
	now := time.Now().Add(delay)
	t.stopSearchTimeout = &now
}

// isStopSearchDelayExpired returns true if the timeout to stop current topic
// search has been accomplished.
func (t *TopicPool) isStopSearchDelayExpired() bool {
	if t.stopSearchTimeout == nil {
		return false
	}
	return t.stopSearchTimeout.Before(time.Now())
}

// readyToStopSearch return true if all conditions to stop search are ok.
func (t *TopicPool) readyToStopSearch() bool {
	return t.isStopSearchDelayExpired() || t.maxCachedPeersReached()
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

	// if selected mode is fast mode and fast mode timeout was not set yet,
	// do it now
	if mode == t.fastMode && t.fastModeTimeoutCancel == nil {
		t.fastModeTimeoutCancel = t.limitFastMode(t.fastModeTimeout)
	}
	// remove fast mode timeout as slow mode is selected now
	if mode == t.slowMode && t.fastModeTimeoutCancel != nil {
		close(t.fastModeTimeoutCancel)
		t.fastModeTimeoutCancel = nil
	}
}

func (t *TopicPool) limitFastMode(timeout time.Duration) chan struct{} {
	if timeout == 0 {
		return nil
	}

	cancel := make(chan struct{})

	t.poolWG.Add(1)
	go func() {
		defer t.poolWG.Done()

		select {
		case <-time.After(timeout):
			t.mu.Lock()
			t.setSyncMode(t.slowMode)
			t.mu.Unlock()
		case <-cancel:
			return
		}
	}()

	return cancel
}

// ConfirmAdded called when peer was added by p2p Server.
// 1. Skip a peer if it not in our peer table
// 2. Add a peer to a cache.
// 3. Disconnect a peer if it was connected after we reached max limit of peers.
//    (we can't know in advance if peer will be connected, thats why we allow
//     to overflow for short duration)
// 4. Switch search to slow mode if it is running.
func (t *TopicPool) ConfirmAdded(server *p2p.Server, nodeID enode.ID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// inbound connection
	peerInfoItem, ok := t.pendingPeers[nodeID]
	if !ok {
		return
	}
	peer := peerInfoItem.peerInfo // get explicit reference

	// established connection means that the node
	// is a viable candidate for a connection and can be cached
	if err := t.cache.AddPeer(peer.node, t.topic); err != nil {
		log.Error("failed to persist a peer", "error", err)
	}

	t.movePeerFromPoolToConnected(nodeID)
	// if the upper limit is already reached, drop this peer
	if len(t.connectedPeers) > t.limits.Max {
		log.Debug("max limit is reached drop the peer", "ID", nodeID, "topic", t.topic)
		peer.dismissed = true
		t.removeServerPeer(server, peer)
		return
	}

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
func (t *TopicPool) ConfirmDropped(server *p2p.Server, nodeID enode.ID) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	// either inbound or connected from another topic
	peer, exist := t.connectedPeers[nodeID]
	if !exist {
		return false
	}

	log.Debug("disconnect", "ID", nodeID, "dismissed", peer.dismissed)

	delete(t.connectedPeers, nodeID)
	// Peer was removed by us because exceeded the limit.
	// Add it back to the pool as it can be useful in the future.
	if peer.dismissed {
		t.addToPendingPeers(peer)
		// use queue for peers that weren't added to p2p server
		t.addToQueue(peer)
		return false
	}

	// If there was a network error, this event will be received
	// but the peer won't be removed from the static nodes set.
	// That's why we need to call `removeServerPeer` manually.
	t.removeServerPeer(server, peer)

	if err := t.cache.RemovePeer(nodeID, t.topic); err != nil {
		log.Error("failed to remove peer from cache", "error", err)
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
	peer := t.popFromQueue()
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
	if !t.discovery.Running() {
		return ErrDiscv5NotRunning
	}
	atomic.StoreInt32(&t.running, 1)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.quit = make(chan struct{})
	t.stopSearchTimeout = nil

	// `period` is used to notify about the current sync mode.
	t.period = make(chan time.Duration, 2)
	// use fast sync mode at the beginning
	t.setSyncMode(t.fastMode)

	// peers management
	found := make(chan *discv5.Node, 5) // 5 reasonable number for concurrently found nodes
	lookup := make(chan bool, 10)       // sufficiently buffered channel, just prevents blocking because of lookup

	for _, peer := range t.cache.GetPeersRange(t.topic, 5) {
		log.Debug("adding a peer from cache", "peer", peer)
		found <- peer
	}

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

	for {
		select {
		case <-t.quit:
			return
		case <-lookup:
		case node := <-found:
			nodeID, err := Discv5IDToEnodeID(node.ID)
			if err != nil {
				continue
			}

			if nodeID != server.Self().ID() {
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
	en, err := Discv5ToEnode(*node)
	if err != nil {
		log.Warn("Failed to convert to enode", "ID", node.ID, "err", err)
		return
	}
	// peer is already connected so update only discoveredTime
	if peer, ok := t.connectedPeers[en.ID()]; ok {
		peer.discoveredTime = mclock.Now()
		return
	}

	if _, ok := t.pendingPeers[en.ID()]; ok {
		t.updatePendingPeer(en.ID(), mclock.Now())
	} else {
		t.addToPendingPeers(&peerInfo{
			discoveredTime: mclock.Now(),
			node:           node,
		})
	}

	// the upper limit is not reached, so let's add this peer
	if len(t.connectedPeers) < t.maxCachedPeers {
		t.addServerPeer(server, t.pendingPeers[en.ID()].peerInfo)
	} else {
		t.addToQueue(t.pendingPeers[en.ID()].peerInfo)
	}
}

func (t *TopicPool) addServerPeer(server *p2p.Server, info *peerInfo) {
	key, err := decodePubkey64(info.node.ID[:])
	if err != nil {
		log.Warn("Failed to decode key", "ID", info.node.ID, "err", err)
		return
	}
	server.AddPeer(enode.NewV4(
		key,
		info.node.IP,
		int(info.node.UDP),
		int(info.node.TCP),
	))
}

func (t *TopicPool) removeServerPeer(server *p2p.Server, info *peerInfo) {
	key, err := decodePubkey64(info.node.ID[:])
	if err != nil {
		log.Warn("Failed to decode key", "ID", info.node.ID, "err", err)
		return
	}
	server.RemovePeer(enode.NewV4(
		key,
		info.node.IP,
		int(info.node.UDP),
		int(info.node.TCP),
	))
}

func (t *TopicPool) isStopped() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.currentMode == 0
}

// StopSearch stops the closes stop
func (t *TopicPool) StopSearch(server *p2p.Server) {
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
	}
	log.Debug("stoping search", "topic", t.topic)
	close(t.quit)
	t.mu.Lock()
	if t.fastModeTimeoutCancel != nil {
		close(t.fastModeTimeoutCancel)
		t.fastModeTimeoutCancel = nil
	}
	t.currentMode = 0
	t.mu.Unlock()
	// wait for poolWG to exit because it writes to period channel
	t.poolWG.Wait()
	close(t.period)
	t.discWG.Wait()
}

// Topic exposes the internal discovery topic.
func (t *TopicPool) Topic() discv5.Topic {
	return t.topic
}

// SetLimits set the limits for the current TopicPool.
func (t *TopicPool) SetLimits(limits params.Limits) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.limits = limits
}

func Discv5ToEnode(n discv5.Node) (*enode.Node, error) {
	pubkey, err := decodePubkey64(n.ID[:])
	if err != nil {
		return &enode.Node{}, err
	}
	return enode.NewV4(pubkey, n.IP, int(n.TCP), int(n.UDP)), nil
}

func Discv5IDToEnodeID(nodeID discv5.NodeID) (enode.ID, error) {
	pubkey, err := decodePubkey64(nodeID[:])
	if err != nil {
		return enode.ID{}, err
	}
	return enode.PubkeyToIDV4(pubkey), nil
}

func EnodeToDiscv5(node enode.Node) *discv5.Node {
	return discv5.NewNode(discv5.PubkeyID(node.Pubkey()), node.IP(), uint16(node.UDP()), uint16(node.TCP()))
}

func decodePubkey64(b []byte) (*ecdsa.PublicKey, error) {
	return crypto.UnmarshalPubkey(append([]byte{0x04}, b...))
}
