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

var (
	ErrPeerPoolStarted    = errors.New("PeerPool is already started")
	ErrPeerPoolNotStarted = errors.New("PeerPool is not started")
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

type PeerPool struct {
	sync.RWMutex

	topics           []TopicPool
	topicToTopicPool map[discv5.Topic]TopicPool
	nodeIDToPeerInfo map[discover.NodeID]*peerInfo

	cache *peers.Cache

	addPeerReq chan addPeerReq

	serverSubscription event.Subscription
	events             chan *p2p.PeerEvent

	wg   sync.WaitGroup
	quit chan struct{}
}

func NewPeerPool(topics []TopicPool, cache *peers.Cache) *PeerPool {
	topicToTopicPool := make(map[discv5.Topic]TopicPool)
	for _, t := range topics {
		topicToTopicPool[t.Topic()] = t
	}

	return &PeerPool{
		topics:           topics,
		topicToTopicPool: topicToTopicPool,
		nodeIDToPeerInfo: make(map[discover.NodeID]*peerInfo),
		cache:            cache,
	}
}

func (p *PeerPool) Topics() []TopicPool {
	p.RLock()
	defer p.RUnlock()

	return p.topics
}

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

	return
}

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
		log.Debug("PeerPool handling unknown peer", "nodeID", nodeID)
		return
	}

	if err := p.addPeerToCache(nodeID); err != nil {
		log.Error("failed to add peer to cache", "nodeID", nodeID, "err", err)
	}

	var confirmed bool

	for _, topic := range peerInfo.topics {
		topicPool := p.topicToTopicPool[topic]
		if err := topicPool.ConfirmAdded(nodeID); err != nil {
			log.Error("failed to confirm added peer", "topic", topic, "err", err)
		} else {
			confirmed = true
		}
	}

	log.Debug("PeerPool handling a peer", "nodeID", nodeID, "confirmed", confirmed)

	if !confirmed {
		peerInfo.dismissed = true

		node := peerInfo.node
		server.RemovePeer(discover.NewNode(nodeID, node.IP, node.UDP, node.TCP))
	}
}

func (p *PeerPool) handleDroppedPeer(server p2pServer, nodeID discover.NodeID) {
	peerInfo, ok := p.nodeIDToPeerInfo[nodeID]
	if !ok {
		return
	}

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
