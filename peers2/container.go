package peers2

import (
	"sync"
	"time"

	"github.com/status-im/status-go/peers"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/discovery"
)

// SetFastSlowDiscoverPeriod sets an instance of `fastSlowDiscoverPeriod`.
func SetFastSlowDiscoverPeriod(p *fastSlowDiscoverPeriod) func(*DiscoveryContainer) {
	return func(c *DiscoveryContainer) {
		c.period = p
	}
}

// SetFastSyncTimeout sets a timeout after which DiscoveryContainer switches to slow peers discovery mode.
func SetFastSyncTimeout(t time.Duration) func(*DiscoveryContainer) {
	return func(c *DiscoveryContainer) {
		c.fastSyncTimeout = t
	}
}

// DiscoveryContainer is an utility structure that wrapps
// discovery related objects. It provides an interface to
// control the whole discovery system.
type DiscoveryContainer struct {
	sync.RWMutex

	discovery       discovery.Discovery
	peerPool        *PeerPool
	topics          []TopicPool
	period          *fastSlowDiscoverPeriod
	fastSyncTimeout time.Duration

	// to prevent running multiple goroutines
	// switching to slow sync
	switchSlowSyncCancel chan struct{}

	wg   sync.WaitGroup
	quit chan struct{}
}

// NewDiscoveryContainer returns a new DiscoveryContainer instance.
func NewDiscoveryContainer(
	d discovery.Discovery, topics []TopicPool, cache *peers.Cache, opts ...func(*DiscoveryContainer),
) *DiscoveryContainer {
	c := &DiscoveryContainer{
		discovery:       d,
		peerPool:        NewPeerPool(topics, cache),
		topics:          topics,
		period:          defaultFastSlowDiscoverPeriod(),
		fastSyncTimeout: defaultFastSyncTimeout,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Start starts all discovery related structures.
// If timeout is larger than 0, the system times out
// after this duration.
func (c *DiscoveryContainer) Start(server *p2p.Server, timeout time.Duration) (err error) {
	if c.quit != nil {
		return nil
	}

	// The order is important: PeerPool, Discovery, Topics.
	c.peerPool.Start(server)
	defer func() {
		if err != nil {
			c.peerPool.Stop()
		}
	}()
	err = c.startDiscovery()
	if err != nil {
		return
	}
	c.startTopics()

	c.quit = make(chan struct{})

	// transition to slow mode after a while
	go c.switchToSlowSync(c.fastSyncTimeout, c.quit)
	// stop Discovery after some time
	if timeout > 0 {
		go c.handleDiscoveryTimeout(timeout, c.quit)
	}
	// Periodically check satisfaction of TopicPools.
	// It might be possible to switch to slower mode faster.
	go c.checkTopicsSatisfaction(time.Second, c.quit)

	return nil
}

// Stop stops all peers discovery system components.
func (c *DiscoveryContainer) Stop() (err error) {
	if c.quit == nil {
		return nil
	}

	close(c.quit)
	c.wg.Wait()

	c.period.close()

	err = c.stopDiscovery()
	c.stopTopics()
	c.peerPool.Stop()

	c.quit = nil

	return
}

func (c *DiscoveryContainer) startDiscovery() (err error) {
	if c.discovery.Running() {
		return nil
	}
	return c.discovery.Start()
}

func (c *DiscoveryContainer) stopDiscovery() error {
	if !c.discovery.Running() {
		return nil
	}
	return c.discovery.Stop()
}

func (c *DiscoveryContainer) startTopics() {
	for _, t := range c.topics {
		t.Start(c.peerPool, c.period.channel())
	}
}

func (c *DiscoveryContainer) stopTopics() {
	for _, t := range c.topics {
		t.Stop()
	}
}

func (c *DiscoveryContainer) handleDiscoveryTimeout(timeout time.Duration, quit <-chan struct{}) {
	c.wg.Add(1)
	defer c.wg.Done()

	select {
	case <-quit:
	case <-time.After(timeout):
		c.RLock()
		if err := c.stopDiscovery(); err != nil {
			log.Error("failed to stop peers discovery container", "err", err)
		}
		c.stopTopics()
		c.RUnlock()
	}
}

// checkTopicsSatisfaction monitors if Discovery and
// TopicPools should be active or can be stopped.
// PeerPool should never be stopped as it watches the peers.
func (c *DiscoveryContainer) checkTopicsSatisfaction(period time.Duration, quit <-chan struct{}) {
	c.wg.Add(1)
	defer c.wg.Done()

	t := time.NewTicker(period)
	defer t.Stop()

	for {
		select {
		case <-quit:
			return
		case <-t.C:
			c.RLock()
			if IsAllTopicsSatisfied(c.peerPool.Topics()) {
				log.Debug("all topics are satisfied")
				if err := c.stopDiscovery(); err != nil {
					log.Error("failed to stop discovery and topics", "err", err)
				}
				c.stopTopics()
			} else {
				log.Debug("not all topics are satisfied")

				// When transitioning from stopped to running Discovery,
				// we should switch to fast mode.
				if !c.discovery.Running() {
					c.period.transFast()
				}

				if err := c.startDiscovery(); err != nil {
					log.Error("failed to start discovery and topics", "err", err)
				} else {
					c.startTopics()
					go c.switchToSlowSync(c.fastSyncTimeout, c.quit)
				}
			}
			c.RUnlock()
		}
	}
}

func (c *DiscoveryContainer) switchToSlowSync(timeout time.Duration, quit <-chan struct{}) {
	c.Lock()
	c.wg.Add(1)
	defer c.wg.Done()

	// cancel the previous schedule switching to slow sync
	if c.switchSlowSyncCancel != nil {
		close(c.switchSlowSyncCancel)
	}
	cancel := make(chan struct{})
	c.switchSlowSyncCancel = cancel

	c.Unlock()

	select {
	case <-quit:
	case <-cancel:
	case <-time.After(timeout):
		c.RLock()
		c.period.transSlow()
		c.RUnlock()
	}
}
