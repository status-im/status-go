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

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
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
	DefaultSlowSync = 30 * time.Second
	// DefaultDiscV5Timeout is a timeout after which Discv5 is stopped.
	DefaultDiscV5Timeout = 3 * time.Minute
	// DefaultTopicFastModeTimeout is a timeout after which sync mode is switched to slow mode.
	DefaultTopicFastModeTimeout = 30 * time.Second
	// DefaultTopicStopSearchDelay is the default delay when stopping a topic search.
	DefaultTopicStopSearchDelay = 10 * time.Second
)

// Options is a struct with PeerPool configuration.
type Options struct {
	FastSync time.Duration
	SlowSync time.Duration
	// After this time, Discovery is stopped even if max peers is not reached.
	DiscServerTimeout time.Duration
	// AllowStop allows stopping Discovery when reaching max peers or after timeout.
	AllowStop bool
	// TopicStopSearchDelay time stopSearch will be waiting for max cached peers to be
	// filled before really stopping the search.
	TopicStopSearchDelay time.Duration
}

// NewDefaultOptions returns a struct with default Options.
func NewDefaultOptions() *Options {
	return &Options{
		FastSync:             DefaultFastSync,
		SlowSync:             DefaultSlowSync,
		DiscServerTimeout:    DefaultDiscV5Timeout,
		AllowStop:            false,
		TopicStopSearchDelay: DefaultTopicStopSearchDelay,
	}
}

type peerInfo struct {
	// discoveredTime last time when node was found by v5
	discoveredTime mclock.AbsTime
	// dismissed is true when our node requested a disconnect
	dismissed bool

	node *discv5.Node
}

// PeerPool manages discovered peers and connects them to p2p server
type PeerPool struct {
	opts *Options

	discovery Discovery

	// config can be set only once per pool life cycle
	config map[discv5.Topic]params.Limits
	cache  *Cache

	mu                 sync.RWMutex
	topics             []TopicPoolInterface
	serverSubscription event.Subscription
	events             chan *p2p.PeerEvent
	quit               chan struct{}
	wg                 sync.WaitGroup
	timeout            <-chan time.Time
}

// NewPeerPool creates instance of PeerPool
func NewPeerPool(discovery Discovery, config map[discv5.Topic]params.Limits, cache *Cache, options *Options) *PeerPool {
	return &PeerPool{
		opts:      options,
		discovery: discovery,
		config:    config,
		cache:     cache,
	}
}

func (p *PeerPool) setDiscoveryTimeout() {
	if p.opts.AllowStop && p.opts.DiscServerTimeout > 0 {
		p.timeout = time.After(p.opts.DiscServerTimeout)
	}
}

// Start creates topic pool for each topic in config and subscribes to server events.
func (p *PeerPool) Start(server *p2p.Server) error {
	if !p.discovery.Running() {
		return ErrDiscv5NotRunning
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// init channels
	p.quit = make(chan struct{})
	p.setDiscoveryTimeout()

	// subscribe to peer events
	p.events = make(chan *p2p.PeerEvent, 20)
	p.serverSubscription = server.SubscribeEvents(p.events)
	p.wg.Add(1)
	go func() {
		p.handleServerPeers(server, p.events)
		p.wg.Done()
	}()

	// collect topics and start searching for nodes
	p.topics = make([]TopicPoolInterface, 0, len(p.config))
	for topic, limits := range p.config {
		var topicPool TopicPoolInterface
		if topic == MailServerDiscoveryTopic {
			topicPool = newCacheOnlyTopicPool(p.discovery, topic, limits, p.opts.SlowSync, p.opts.FastSync, p.cache)
		} else {
			topicPool = newTopicPool(p.discovery, topic, limits, p.opts.SlowSync, p.opts.FastSync, p.cache)
		}
		if err := topicPool.StartSearch(server); err != nil {
			return err
		}
		p.topics = append(p.topics, topicPool)
	}

	// discovery must be already started when pool is started
	signal.SendDiscoveryStarted()

	return nil
}

func (p *PeerPool) startDiscovery() error {
	if p.discovery.Running() {
		return nil
	}

	if err := p.discovery.Start(); err != nil {
		return err
	}

	p.mu.Lock()
	p.setDiscoveryTimeout()
	p.mu.Unlock()

	signal.SendDiscoveryStarted()

	return nil
}

func (p *PeerPool) stopDiscovery() {
	if !p.discovery.Running() {
		return
	}

	for _, t := range p.topics {
		t.StopSearch()
	}
	if err := p.discovery.Stop(); err != nil {
		log.Error("discovery errored when was closed", "err", err)
	}

	p.mu.Lock()
	p.timeout = nil
	p.mu.Unlock()

	signal.SendDiscoveryStopped()
}

// restartDiscovery and search for topics that have peer count below min
func (p *PeerPool) restartDiscovery(server *p2p.Server) error {
	if !p.discovery.Running() {
		if err := p.startDiscovery(); err != nil {
			return err
		}
		log.Debug("restarted discovery from peer pool")
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
//
// @TODO(adam): split it into peers and discovery management loops. This should
// simplify the whole logic and allow to remove `timeout` field from `PeerPool`.
func (p *PeerPool) handleServerPeers(server *p2p.Server, events <-chan *p2p.PeerEvent) {
	var retryDiscv5 <-chan time.Time
	var stopDiscv5 <-chan time.Time

	for {
		p.mu.RLock()
		timeout := p.timeout
		p.mu.RUnlock()

		select {
		case <-p.quit:
			log.Debug("stopping DiscV5 because of quit")
			p.stopDiscovery()
			return
		case <-timeout:
			log.Info("DiscV5 timed out")
			p.stopDiscovery()
		case <-retryDiscv5:
			if err := p.restartDiscovery(server); err != nil {
				retryDiscv5 = time.After(discoveryRestartTimeout)
				log.Error("starting discv5 failed", "error", err, "retry", discoveryRestartTimeout)
			}
		case <-stopDiscv5:
			p.handleStopTopics()
		case event := <-events:
			switch event.Type {
			case p2p.PeerEventTypeDrop:
				log.Debug("confirm peer dropped", "ID", event.Peer)
				if p.handleDroppedPeer(server, event.Peer) {
					retryDiscv5 = time.After(0)
				}
			case p2p.PeerEventTypeAdd: // skip other events
				log.Debug("confirm peer added", "ID", event.Peer)
				p.handleAddedPeer(server, event.Peer)
				stopDiscv5 = time.After(p.opts.TopicStopSearchDelay)
			default:
				continue
			}
			SendDiscoverySummary(server.PeersInfo())
		}
	}
}

// handleAddedPeer notifies all topics about added peer.
func (p *PeerPool) handleAddedPeer(server *p2p.Server, nodeID discover.NodeID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, t := range p.topics {
		t.ConfirmAdded(server, nodeID)
		if p.opts.AllowStop && t.MaxReached() {
			t.setStopSearchTimeout(p.opts.TopicStopSearchDelay)
		}
	}
}

// handleStopTopics stops the search on any topics having reached its max cached
// limit or its delay stop is expired, additionally will stop discovery if all
// peers are stopped.
func (p *PeerPool) handleStopTopics() {
	if !p.opts.AllowStop {
		return
	}
	for _, t := range p.topics {
		if t.readyToStopSearch() {
			t.StopSearch()
		}
	}
	if p.allTopicsStopped() {
		log.Debug("closing discv5 connection because all topics reached max limit")
		p.stopDiscovery()
	}
}

// allTopicsStopped returns true if all topics are stopped.
func (p *PeerPool) allTopicsStopped() (all bool) {
	if !p.opts.AllowStop {
		return false
	}
	all = true
	for _, t := range p.topics {
		if !t.isStopped() {
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
		log.Debug("search", "topic", t.Topic(), "below min", t.BelowMin())
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
	p.wg.Wait()
}
