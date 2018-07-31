package peers2

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/peers"
)

type p2pServer interface {
	SubscribeEvents(chan *p2p.PeerEvent) event.Subscription
	Self() *discover.Node
	PeersInfo() []*p2p.PeerInfo
	AddPeer(*discover.Node)
	RemovePeer(*discover.Node)
}

type addPeerReq struct {
	t    discv5.Topic
	node *discv5.Node
}

type peerInfo struct {
	topics    []discv5.Topic
	node      *discv5.Node
	dismissed bool
}

// PeerPool manages found peers by `TopicPool`s and calls appropriate methods
// when founds peers are connected or dropped.PeerPool
// It also manages peers caching.
type PeerPool struct {
	sync.RWMutex

	topicToTopicPool map[discv5.Topic]TopicPool
	nodeIDToPeerInfo map[discover.NodeID]*peerInfo

	cache *peers.Cache

	addPeerReq chan addPeerReq

	serverSubscription event.Subscription
	events             chan *p2p.PeerEvent

	wg   sync.WaitGroup
	quit chan struct{}
}

// NewPeerPool returns a new PeerPool instance.
func NewPeerPool(topics []TopicPool, cache *peers.Cache) *PeerPool {
	topicToTopicPool := make(map[discv5.Topic]TopicPool)
	for _, t := range topics {
		topicToTopicPool[t.Topic()] = t
	}

	return &PeerPool{
		topicToTopicPool: topicToTopicPool,
		nodeIDToPeerInfo: make(map[discover.NodeID]*peerInfo),
		cache:            cache,
	}
}

// Topics returns a list of followed `TopicPool`s.
func (p *PeerPool) Topics() []TopicPool {
	p.RLock()
	defer p.RUnlock()

	topics := make([]TopicPool, 0, len(p.topicToTopicPool))
	for _, t := range p.topicToTopicPool {
		topics = append(topics, t)
	}

	return topics
}

// Start starts the `PeerPool` mechanics.
// This method is idempotent and if the `PeerPool`
// is already started, it returns immediatelly.
func (p *PeerPool) Start(server p2pServer) {
	p.Lock()
	defer p.Unlock()

	if p.quit != nil {
		return
	}
	p.quit = make(chan struct{})

	// subscribe to peer events
	p.events = make(chan *p2p.PeerEvent, 20)
	p.serverSubscription = server.SubscribeEvents(p.events)

	p.addPeerReq = make(chan addPeerReq)
	go p.handleServerPeers(server, p.events, p.addPeerReq)

	// load initial peers from cache
	p.loadInitialPeersFromCache()

	return
}

// Stop stops the `PeerPool`.
func (p *PeerPool) Stop() {
	p.Lock()
	defer p.Unlock()

	if p.quit == nil {
		return
	}

	select {
	case <-p.quit:
		return
	default:
		close(p.quit)
	}

	p.serverSubscription.Unsubscribe()
	p.wg.Wait()

	p.quit = nil
}

// RequestToAddPeer passes along a request to add a node as a peer.
func (p *PeerPool) RequestToAddPeer(t discv5.Topic, node *discv5.Node) {
	p.RLock()
	p.addPeerReq <- addPeerReq{t, node}
	p.RUnlock()
}

func (p *PeerPool) handleServerPeers(
	server p2pServer, events <-chan *p2p.PeerEvent, addPeerReq <-chan addPeerReq,
) {
	p.wg.Add(1)
	defer p.wg.Done()

	selfID := server.Self().ID

	for {
		select {
		case <-p.quit:
			return
		case event := <-events:
			switch event.Type {
			case p2p.PeerEventTypeAdd:
				p.handleAddedPeer(server, event.Peer)
			case p2p.PeerEventTypeDrop:
				p.handleDroppedPeer(server, event.Peer)
			default:
				continue
			}
			peers.SendDiscoverySummary(server.PeersInfo())
		case req := <-addPeerReq:
			t := req.t
			node := req.node
			nodeID := discover.NodeID(node.ID)

			if nodeID == selfID {
				continue
			}

			if _, ok := p.nodeIDToPeerInfo[nodeID]; !ok {
				p.nodeIDToPeerInfo[nodeID] = &peerInfo{}
			}

			peerInfo := p.nodeIDToPeerInfo[nodeID]
			peerInfo.topics = append(peerInfo.topics, t)
			peerInfo.node = node

			log.Debug("adding peer to server", "nodeID", nodeID, "IP", node.IP, "TPC", node.TCP, "UDP", node.UDP)
			server.AddPeer(discover.NewNode(nodeID, node.IP, node.UDP, node.UDP))
		}
	}
}

func (p *PeerPool) handleAddedPeer(server p2pServer, nodeID discover.NodeID) {
	peerInfo, ok := p.nodeIDToPeerInfo[nodeID]
	if !ok {
		log.Debug("PeerPool adding an unknown peer", "nodeID", nodeID)
		return
	}

	if err := p.addPeerToCache(nodeID); err != nil {
		log.Error("failed to add peer to cache", "nodeID", nodeID, "err", err)
	}

	// Check if a given peer gets confirmed. It must receive at least one confirmation
	// from a `TopicPool` in order not to be dropped.
	var confirmed bool

	for _, topic := range peerInfo.topics {
		topicPool := p.topicToTopicPool[topic]
		if err := topicPool.ConfirmAdded(nodeID); err != nil {
			log.Error("failed to confirm added peer", "topic", topic, "err", err)
		} else {
			confirmed = true
		}
	}

	log.Debug("PeerPool adding a peer", "nodeID", nodeID, "confirmed", confirmed)

	if !confirmed {
		// indicate that the `PeerPool` dropped this peer deliberetly
		peerInfo.dismissed = true

		node := peerInfo.node
		server.RemovePeer(discover.NewNode(nodeID, node.IP, node.UDP, node.TCP))
	}
}

func (p *PeerPool) handleDroppedPeer(server p2pServer, nodeID discover.NodeID) {
	peerInfo, ok := p.nodeIDToPeerInfo[nodeID]
	if !ok {
		log.Debug("PeerPool dropping an unknown peer", "nodeID", nodeID)
		return
	}

	// Peer is not removed from the cache if it was dropped by the `PeerPool` deliberetly.
	// It means that the peer is ok but no topic is interested in it anymore.
	if !peerInfo.dismissed {
		if err := p.removePeerFromCache(nodeID); err != nil {
			log.Error("failed to remove peer from cache", "nodeID", nodeID, "err", err)
		}
	}

	for _, topic := range peerInfo.topics {
		topicPool := p.topicToTopicPool[topic]
		if err := topicPool.ConfirmDropped(nodeID); err != nil {
			log.Error("failed to confirm dropped peer", "topic", topic, "err", err)
		}
	}

	delete(p.nodeIDToPeerInfo, nodeID)
}

func (p *PeerPool) loadInitialPeersFromCache() {
	for topic, topicPool := range p.topicToTopicPool {
		limit := 5
		if tp, ok := topicPool.(limited); ok {
			limit = tp.UpperLimit()
		}

		nodes := p.getPeersFromCache(topic, limit)
		for _, node := range nodes {
			p.addPeerReq <- addPeerReq{topic, node}
		}
	}
}

func (p *PeerPool) getPeersFromCache(topic discv5.Topic, limit int) []*discv5.Node {
	if p.cache == nil {
		return nil
	}

	return p.cache.GetPeersRange(topic, limit)
}

func (p *PeerPool) addPeerToCache(nodeID discover.NodeID) error {
	if p.cache == nil {
		return nil
	}

	peerInfo, ok := p.nodeIDToPeerInfo[nodeID]
	if !ok {
		return errors.New("nodeID node found")
	}

	for _, t := range peerInfo.topics {
		if err := p.cache.AddPeer(peerInfo.node, t); err != nil {
			log.Error("failed to add peer to cache", "nodeID", nodeID, "topic", t)
		}
	}

	return nil
}

func (p *PeerPool) removePeerFromCache(nodeID discover.NodeID) error {
	if p.cache == nil {
		return nil
	}

	peerInfo, ok := p.nodeIDToPeerInfo[nodeID]
	if !ok {
		return errors.New("nodeID node found")
	}

	for _, t := range peerInfo.topics {
		if err := p.cache.RemovePeer(discv5.NodeID(nodeID), t); err != nil {
			log.Error("failed to remove peer from cache", "nodeID", nodeID, "topic", t)
		}
	}

	return nil
}
