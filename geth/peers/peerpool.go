package peers

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"

	"github.com/status-im/status-go/geth/params"
)

var (
	// ErrDiscv5NotRunning returned when pool is started but discover v5 is not running or not enabled.
	ErrDiscv5NotRunning = errors.New("Discovery v5 is not running")
)

// PoolEvent is a type used to for peer pool events.
type PoolEvent string

const (
	// expirationPeriod is an amount of time while peer is considered as a connectable
	expirationPeriod = 60 * time.Minute
	// discoveryRestartTimeout defines how often loop will try to start discovery server
	discoveryRestartTimeout = 2 * time.Second
	// DefaultFastSync is a recommended value for aggressive peers search.
	DefaultFastSync = 3 * time.Second
	// DefaultSlowSync is a recommended value for slow (background) peers search.
	DefaultSlowSync = 30 * time.Minute

	// Discv5Closed is sent when discv5 is closed
	Discv5Closed PoolEvent = "discv5.closed"
	// Discv5Started is sent when discv5 is started
	Discv5Started PoolEvent = "discv5.started"
)

// NewPeerPool creates instance of PeerPool
func NewPeerPool(config map[discv5.Topic]params.Limits, fastSync, slowSync time.Duration, cache *Cache, stopOnMax bool) *PeerPool {
	return &PeerPool{
		config:    config,
		fastSync:  fastSync,
		slowSync:  slowSync,
		cache:     cache,
		stopOnMax: stopOnMax,
	}
}

type peerInfo struct {
	// discoveredTime last time when node was found by v5
	discoveredTime mclock.AbsTime
	// connected is true if node is added as a static peer
	connected bool
	// requested is true when our node requested a disconnect
	requested bool

	node *discv5.Node
}

// PeerPool manages discovered peers and connects them to p2p server
type PeerPool struct {
	// config can be set only once per pool life cycle
	config    map[discv5.Topic]params.Limits
	fastSync  time.Duration
	slowSync  time.Duration
	cache     *Cache
	stopOnMax bool

	mu                 sync.RWMutex
	topics             []*TopicPool
	serverSubscription event.Subscription
	quit               chan struct{}

	wg sync.WaitGroup

	feed event.Feed
}

// Start creates topic pool for each topic in config and subscribes to server events.
func (p *PeerPool) Start(server *p2p.Server) error {
	if server.DiscV5 == nil {
		return ErrDiscv5NotRunning
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.quit = make(chan struct{})
	p.topics = make([]*TopicPool, 0, len(p.config))
	for topic, limits := range p.config {
		topicPool := NewTopicPool(topic, limits, p.slowSync, p.fastSync)
		if err := topicPool.StartSearch(server); err != nil {
			return err
		}
		p.topics = append(p.topics, topicPool)
	}

	events := make(chan *p2p.PeerEvent, 20)
	p.serverSubscription = server.SubscribeEvents(events)
	p.wg.Add(1)
	go func() {
		p.handleServerPeers(server, events)
		p.wg.Done()
	}()
	return nil
}

// restartDiscovery and search for topics that have peer count below min
func (p *PeerPool) restartDiscovery(server *p2p.Server) error {
	if server.DiscV5 == nil {
		ntab, err := StartDiscv5(server)
		if err != nil {
			log.Error("starting discv5 failed", "error", err, "retry in", discoveryRestartTimeout)
			return err
		}
		log.Debug("restarted discovery from peer pool")
		server.DiscV5 = ntab
		p.feed.Send(Discv5Started)
	}
	for _, t := range p.topics {
		if !t.BelowMin() || t.SearchRunning() {
			continue
		}
		err := t.StartSearch(server)
		if err != nil {
			log.Error("search failed to start", "error", err)
		}
	}
	return nil
}

// handleServerPeers watches server peer events, notifies topic pools about changes
// in the peer set and stops the discv5 if all topic pools collected enough peers.
func (p *PeerPool) handleServerPeers(server *p2p.Server, events <-chan *p2p.PeerEvent) {
	var retryDiscv5 <-chan time.Time

	for {
		select {
		case <-p.quit:
			return
		case <-retryDiscv5:
			if err := p.restartDiscovery(server); err != nil {
				retryDiscv5 = time.After(discoveryRestartTimeout)
			}
		case event := <-events:
			switch event.Type {
			case p2p.PeerEventTypeDrop:
				log.Debug("confirm peer dropped", "ID", event.Peer)
				if p.stopOnMax && p.handleDroppedPeer(server, event.Peer) {
					retryDiscv5 = time.After(0)
				}
			case p2p.PeerEventTypeAdd:
				log.Debug("confirm peer added", "ID", event.Peer)
				if p.stopOnMax && p.handleAddedPeer(server, event.Peer) {
					log.Debug("closing discv5 connection", "server", server.Self())
					server.DiscV5.Close()
					server.DiscV5 = nil
					p.feed.Send(Discv5Closed)
				}
			}
		}
	}
}

// handleAddedPeer notifies all topics about added peer and return true if all topics has max limit of connections
func (p *PeerPool) handleAddedPeer(server *p2p.Server, nodeID discover.NodeID) (all bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	all = true
	for _, t := range p.topics {
		t.ConfirmAdded(server, nodeID)
		if p.stopOnMax && t.MaxReached() {
			t.StopSearch()
		} else {
			all = false
		}
	}
	return all
}

// handleDroppedPeer notifies every topic about dropped peer and returns true if any peer have connections
// below min limit
func (p *PeerPool) handleDroppedPeer(server *p2p.Server, nodeID discover.NodeID) (any bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, t := range p.topics {
		confirmed := t.ConfirmDropped(server, nodeID)
		if confirmed {
			newPeer := t.AddPeerFromTable(server)
			if newPeer != nil {
				log.Debug("added peer from local table", "ID", newPeer.ID)
			}
		}
		log.Debug("search", "topic", t.topic, "below min", t.BelowMin())
		if t.BelowMin() && !t.SearchRunning() {
			any = true
		}
	}
	return any
}

// Stop closes pool quit channel and all channels that are watched by search queries
// and waits till all goroutines will exit.
func (p *PeerPool) Stop() {
	// pool wasn't started
	if p.quit == nil {
		return
	}
	select {
	case <-p.quit:
		return
	default:
		log.Debug("started closing peer pool")
		close(p.quit)
	}
	p.serverSubscription.Unsubscribe()
	// wait before closing topic pools, otherwise there is chance that
	// they will be concurrently started while we are exiting.
	p.wg.Wait()
	for _, t := range p.topics {
		t.StopSearch()
	}
}
