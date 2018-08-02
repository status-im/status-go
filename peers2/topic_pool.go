package peers2

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/discovery"
)

type peerID = discover.NodeID

// TopicPool discovers peers using Discovery interface
// and confirms if an added peer is valid.
//
// `ConfirmAdded` can return an error if a peer is not valid anymore.
type TopicPool interface {
	Topic() discv5.Topic
	Start(*PeerPool, <-chan time.Duration)
	Stop()
	ConfirmAdded(peerID) error
	ConfirmDropped(peerID) error
}

type satisfiable interface {
	Satisfied() bool
}

type limited interface {
	UpperLimit() int
}

// IsTopicSatisfied returns true if a given `TopicPool` is satisfied
// with currently connected peers.
// If `TopicPool` does not implement a `satisfiable` interface,
// it returns true by default.
func IsTopicSatisfied(topic TopicPool) bool {
	st, ok := topic.(satisfiable)
	if !ok {
		return true
	}

	return st.Satisfied()
}

// IsAllTopicsSatisfied checks if all `TopicPool`s are satisfied.
func IsAllTopicsSatisfied(topics []TopicPool) bool {
	for _, topic := range topics {
		if !IsTopicSatisfied(topic) {
			return false
		}
	}
	return true
}

// SetDiscoverPeriod sets a period for sending a discover peers request.
func SetDiscoverPeriod(p time.Duration) func(*TopicPoolBase) {
	return func(t *TopicPoolBase) {
		periodCh := make(chan time.Duration, 1)
		periodCh <- p
		t.period = periodCh
	}
}

// SetPeersHandler sets a handler which verifies each found peer.
func SetPeersHandler(h FoundPeersHandler) func(*TopicPoolBase) {
	return func(t *TopicPoolBase) {
		t.peersHandler = h
	}
}

// TopicPoolBase is a minimal implementation of `TopicPool`.
type TopicPoolBase struct {
	sync.RWMutex

	discovery    discovery.Discovery
	topic        discv5.Topic
	period       <-chan time.Duration
	peersHandler FoundPeersHandler

	handlerDone  <-chan struct{}
	discoverDone <-chan struct{}
	quit         chan struct{}
}

var _ TopicPool = (*TopicPoolBase)(nil)

// NewTopicPoolBase creates a new instance of `TopicPoolBase`.
func NewTopicPoolBase(
	d discovery.Discovery, t discv5.Topic, opts ...func(*TopicPoolBase),
) *TopicPoolBase {
	topicPool := &TopicPoolBase{
		discovery: d,
		topic:     t,
	}

	for _, opt := range opts {
		opt(topicPool)
	}

	if topicPool.peersHandler == nil {
		topicPool.peersHandler = &AcceptAllPeersHandler{}
	}

	return topicPool
}

// Topic returns a topic name.
func (t *TopicPoolBase) Topic() discv5.Topic {
	t.RLock()
	defer t.RUnlock()
	return t.topic
}

// Start starts discovering peers for a given topic.
// It also checks if all required parameters are set and
// if not, it will set defaults.
func (t *TopicPoolBase) Start(pool *PeerPool, period <-chan time.Duration) {
	t.Lock()
	defer t.Unlock()

	if t.quit != nil {
		return
	}
	t.quit = make(chan struct{})

	t.period = period

	var (
		found  chan *discv5.Node
		lookup <-chan bool
	)
	found, lookup, t.discoverDone = t.discover(t.period)
	t.handlerDone = t.handleFoundPeers(pool, found, lookup)
}

// Stop stops discovering a given topic.
func (t *TopicPoolBase) Stop() {
	if t.quit == nil {
		return
	}

	select {
	case <-t.quit:
		return
	default:
	}

	// Wait for the `discover` method to exit first. Otherwise,
	// it may still be returning nodes while the found peers handler
	// is stopped.
	// `discover` can be cloed only by closing `period`.
	<-t.discoverDone

	close(t.quit)
	// wait for found peers handler method to return
	<-t.handlerDone
	t.quit = nil
}

// ConfirmAdded is called when a found peer was discovered.
// In case of a basic implementation, it does nothing.
func (t *TopicPoolBase) ConfirmAdded(peer peerID) error {
	log.Debug("TopicPoolBase confirming peer added", "topic", t.topic, "peerID", peer)
	return nil
}

// ConfirmDropped is called when a previously found and connected peer
// was dropped.
// In case of a basic implementation, it does nothing.
func (t *TopicPoolBase) ConfirmDropped(peer peerID) error {
	log.Debug("TopicPoolBase confirming peer dropped", "topic", t.topic, "peerID", peer)
	return nil
}

// discover register itself in Discovery and waits for found nodes.
// `found` channel is returned as read/write in order to allow performing some unit tests.
func (t *TopicPoolBase) discover(period <-chan time.Duration) (chan *discv5.Node, <-chan bool, <-chan struct{}) {
	topic := t.topic
	done := make(chan struct{})
	found := make(chan *discv5.Node, 5) // 5 reasonable number for concurrently found nodes
	lookup := make(chan bool, 10)       // sufficiently buffered channel, just prevents blocking because of lookup

	go func() {
		err := t.discovery.Discover(string(topic), period, found, lookup)
		if err != nil {
			// TODO(adam): this should be reported to the caller in order to resurect TopicPool
			log.Error("error searching for", "topic", topic, "err", err)
		}
		close(done)
	}()

	return found, lookup, done
}

func (t *TopicPoolBase) handleFoundPeers(
	pool *PeerPool, found <-chan *discv5.Node, lookup <-chan bool,
) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			select {
			case <-t.quit:
				return
			case <-lookup:
			case node := <-found:
				if t.peersHandler.Handle(node) {
					pool.RequestToAddPeer(t.topic, node)
				}
			}
		}
	}()

	return done
}

// TopicPoolWithLimits handles peers but uses limits to do that.
// If there is more than the upper limit peers connected,
// `ConfirmAdded` returns an error so that the peer can be dropped.
//
// It also implements `satisfiable` interface and it is satisfied
// if at least the lower limit of connected peers is reached.
type TopicPoolWithLimits struct {
	*TopicPoolBase

	connectedPeers map[peerID]struct{}
	minPeers       int
	maxPeers       int
}

var _ TopicPool = (*TopicPoolWithLimits)(nil)

// NewTopicPoolWithLimits returns a new instance of `TopicPoolWithLimits`.
func NewTopicPoolWithLimits(base *TopicPoolBase, minPeers, maxPeers int) *TopicPoolWithLimits {
	return &TopicPoolWithLimits{
		TopicPoolBase:  base,
		connectedPeers: make(map[peerID]struct{}),
		minPeers:       minPeers,
		maxPeers:       maxPeers,
	}
}

// ConfirmAdded returns error if the number of connected peers exceeds
// the upper limit. Otherwise, it returns `nil`.
func (t *TopicPoolWithLimits) ConfirmAdded(peer peerID) error {
	t.Lock()
	defer t.Unlock()

	log.Debug("TopicPoolWithLimits confirming peer added", "topic", t.topic, "peerID", peer)

	t.connectedPeers[peer] = struct{}{}

	if len(t.connectedPeers) > t.maxPeers {
		return errors.New("the upper limit was reached")
	}

	return nil
}

// ConfirmDropped confirms removal of the peer.
func (t *TopicPoolWithLimits) ConfirmDropped(peer peerID) error {
	log.Debug("confirm peer dropped", "topic", t.topic, "peerID", peer)

	t.Lock()
	delete(t.connectedPeers, peer)
	t.Unlock()

	return nil
}

// UpperLimit returns an upper limit of peers.
func (t *TopicPoolWithLimits) UpperLimit() int {
	return t.maxPeers
}

// Satisfied returns true if the number of connected peers for this topic
// reaches at least the lower limit.
func (t *TopicPoolWithLimits) Satisfied() bool {
	t.RLock()
	defer t.RUnlock()

	return len(t.connectedPeers) >= t.minPeers
}

// TopicPoolEphemeral discards the connected peer immediately.
type TopicPoolEphemeral struct {
	*TopicPoolWithLimits
}

// NewTopicPoolEphemeral returns a new instance of `TopicPoolEphemeral`.
func NewTopicPoolEphemeral(base *TopicPoolWithLimits) *TopicPoolEphemeral {
	return &TopicPoolEphemeral{base}
}

// ConfirmAdded always returns an error that indicates that a peer can be dropped.
func (t *TopicPoolEphemeral) ConfirmAdded(peer peerID) error {
	t.Lock()
	defer t.Unlock()

	// In case of TopicPoolEphemeral, `connectedPeers` contains a number of found peers
	// with confirmed connectivity. Peers are never removed.
	t.connectedPeers[peer] = struct{}{}

	return errors.New("ephemeral topic pool")
}
