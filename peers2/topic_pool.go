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

type TopicPool interface {
	Topic() discv5.Topic
	Start(*PeerPool)
	Stop()
	ConfirmAdded(peerID) error
	ConfirmDropped(peerID) error
}

type satisfiable interface {
	Satisfied() bool
}

func IsTopicSatisfied(topic TopicPool) bool {
	st, ok := topic.(satisfiable)
	if !ok {
		// by default, we assume the topic is satisfied
		return true
	}

	return st.Satisfied()
}

func IsAllTopicsSatisfied(topics []TopicPool) bool {
	for _, topic := range topics {
		st, ok := topic.(satisfiable)
		if !ok {
			continue
		}

		if !st.Satisfied() {
			log.Debug("topic not satisfied", "topic", topic.Topic())
			return false
		}
	}

	return true
}

func SetDiscoverPeriod(p time.Duration) func(*TopicPoolBase) {
	return func(t *TopicPoolBase) {
		periodCh := make(chan time.Duration, 1)
		periodCh <- p
		t.period = periodCh
	}
}

func SetPeersHandler(h FoundPeersHandler) func(*TopicPoolBase) {
	return func(t *TopicPoolBase) {
		t.peersHandler = h
	}
}

type TopicPoolBase struct {
	sync.RWMutex

	discovery    discovery.Discovery
	topic        discv5.Topic
	period       chan time.Duration
	peersHandler FoundPeersHandler

	quit chan struct{}
}

var _ TopicPool = (*TopicPoolBase)(nil)

func NewTopicPoolBase(d discovery.Discovery, t discv5.Topic, opts ...func(*TopicPoolBase)) *TopicPoolBase {
	topicPool := &TopicPoolBase{
		discovery: d,
		topic:     t,
	}

	for _, opt := range opts {
		opt(topicPool)
	}

	return topicPool
}

func (t *TopicPoolBase) Topic() discv5.Topic {
	return t.topic
}

func (t *TopicPoolBase) Start(pool *PeerPool) {
	t.Lock()
	defer t.Unlock()

	if t.quit != nil {
		return
	}
	t.quit = make(chan struct{})

	if t.period == nil {
		t.period = make(chan time.Duration, 1)
		t.period <- time.Second
	}
	if t.peersHandler == nil {
		t.peersHandler = &AcceptAllPeersHandler{}
	}

	found, lookup := t.discover(t.period)
	go t.handleFoundPeers(pool, found, lookup)

	return
}

func (t *TopicPoolBase) Stop() {
	if t.quit == nil {
		return
	}

	select {
	case <-t.quit:
		return
	default:
		close(t.quit)
	}

	close(t.period)
}

func (t *TopicPoolBase) ConfirmAdded(peer peerID) error {
	log.Debug("TopicPoolBase confirming peer added", "topic", t.topic, "peerID", peer)
	return nil
}

func (t *TopicPoolBase) ConfirmDropped(peer peerID) error {
	log.Debug("TopicPoolBase confirming peer dropped", "topic", t.topic, "peerID", peer)
	return nil
}

func (t *TopicPoolBase) discover(period <-chan time.Duration) (<-chan *discv5.Node, <-chan bool) {
	found := make(chan *discv5.Node, 5) // 5 reasonable number for concurrently found nodes
	lookup := make(chan bool, 10)       // sufficiently buffered channel, just prevents blocking because of lookup

	go func() {
		err := t.discovery.Discover(string(t.topic), period, found, lookup)
		if err != nil {
			// TODO(adam): this should be reported to the caller in order to resurect TopicPool
			log.Error("error searching for", "topic", t.topic, "err", err)
		}
	}()

	return found, lookup
}

func (t *TopicPoolBase) handleFoundPeers(
	pool *PeerPool, found <-chan *discv5.Node, lookup <-chan bool,
) {
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
}

type TopicPoolWithLimits struct {
	*TopicPoolBase

	connectedPeers map[peerID]struct{}
	minPeers       int
	maxPeers       int
}

var _ TopicPool = (*TopicPoolWithLimits)(nil)

func NewTopicPoolWithLimits(base *TopicPoolBase, minPeers, maxPeers int) *TopicPoolWithLimits {
	return &TopicPoolWithLimits{
		TopicPoolBase:  base,
		connectedPeers: make(map[peerID]struct{}),
		minPeers:       minPeers,
		maxPeers:       maxPeers,
	}
}

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

func (t *TopicPoolWithLimits) ConfirmDropped(peer peerID) error {
	log.Debug("confirm peer dropped", "topic", t.topic, "peerID", peer)

	t.Lock()
	delete(t.connectedPeers, peer)
	t.Unlock()

	return nil
}

func (t *TopicPoolWithLimits) Satisfied() bool {
	t.RLock()
	defer t.RUnlock()

	return len(t.connectedPeers) >= t.minPeers
}

type TopicPoolEphemeral struct {
	*TopicPoolWithLimits
}

func NewTopicPoolEphemeral(base *TopicPoolWithLimits) *TopicPoolEphemeral {
	return &TopicPoolEphemeral{base}
}

func (t *TopicPoolEphemeral) ConfirmAdded(peer peerID) error {
	return errors.New("ephemeral topic pool")
}
