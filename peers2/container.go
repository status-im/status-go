package peers2

import (
	"sync"
	"time"

	"github.com/status-im/status-go/peers"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/discovery"
)

// DiscoveryContainer is an utility structure that wrapps
// discovery related objects. It provides an interface to
// control the whole discovery system.
type DiscoveryContainer struct {
	discovery discovery.Discovery
	peerPool  *PeerPool
	topics    []TopicPool
	period    *fastSlowDiscoverPeriod

	discoveryRunning bool

	wg   sync.WaitGroup
	quit chan struct{}
}

// NewDiscoveryContainer returns a new DiscoveryContainer instance.
func NewDiscoveryContainer(
	d discovery.Discovery, topics []TopicPool, cache *peers.Cache, period *fastSlowDiscoverPeriod,
) *DiscoveryContainer {
	return &DiscoveryContainer{
		discovery: d,
		peerPool:  NewPeerPool(topics, cache),
		topics:    topics,
		period:    period,
	}
}

// Start starts all discovery related structures.
// If timeout is larger than 0, the system times out
// after this duration.
func (c *DiscoveryContainer) Start(server *p2p.Server, timeout time.Duration) (err error) {
	if c.quit != nil {
		return nil
	}

	c.peerPool.Start(server)
	defer func() {
		if err != nil {
			c.peerPool.Stop()
		}
	}()

	err = c.startDiscoveryAndTopics()
	if err != nil {
		return
	}
	go c.switchToSlowMode()

	c.quit = make(chan struct{})

	if timeout > 0 {
		go c.handleTimeout(time.After(timeout))
	}
	go c.checkTopicsSatisfaction(time.Second)

	return nil
}

// Stop stops all peers discovery system components.
func (c *DiscoveryContainer) Stop() (err error) {
	if c.quit == nil {
		return nil
	}

	close(c.quit)
	c.wg.Wait()

	err = c.stopDiscoveryAndTopics()
	c.peerPool.Stop()

	c.quit = nil

	return
}

func (c *DiscoveryContainer) startDiscoveryAndTopics() error {
	if c.discoveryRunning {
		return nil
	}

	// TODO(adam): can Discovery.Start() be idempotent?
	if err := c.discovery.Start(); err != nil {
		return err
	}
	c.discoveryRunning = true

	for _, t := range c.topics {
		t.Start(c.peerPool)
	}

	return nil
}

func (c *DiscoveryContainer) stopDiscoveryAndTopics() (err error) {
	for _, t := range c.topics {
		t.Stop()
	}

	err = c.discovery.Stop()
	if err == nil {
		c.discoveryRunning = false
	}

	return
}

func (c *DiscoveryContainer) handleTimeout(t <-chan time.Time) {
	c.wg.Add(1)
	defer c.wg.Done()

	select {
	case <-c.quit:
	case <-t:
		if err := c.Stop(); err != nil {
			log.Error("failed to stop peers discovery container", "err", err)
		}
	}
}

// checkTopicsSatisfaction monitors if Discovery and TopicPools should be active
// or can be stopped.
// PeerPool should not be stopped as it watches the peers.
func (c *DiscoveryContainer) checkTopicsSatisfaction(period time.Duration) {
	c.wg.Add(1)
	defer c.wg.Done()

	t := time.NewTicker(period)
	defer t.Stop()

	for {
		select {
		case <-c.quit:
			return
		case <-t.C:
			if IsAllTopicsSatisfied(c.peerPool.Topics()) {
				log.Debug("all topics are satisfied")
				if err := c.stopDiscoveryAndTopics(); err != nil {
					log.Error("failed to stop discovery and topics", "err", err)
				}
			} else {
				log.Debug("not all topics are satisfied")

				// When transitioning from stopped to running,
				// we should switch to fast mode.
				if !c.discoveryRunning {
					c.period.transFast()
				}

				if err := c.startDiscoveryAndTopics(); err != nil {
					log.Error("failed to start discovery and topics", "err", err)
				} else {
					go c.switchToSlowMode()
				}
			}
		}
	}
}

func (c *DiscoveryContainer) switchToSlowMode() {
	select {
	case <-c.quit:
		return
	case <-time.After(time.Second):
		c.period.transSlow()
	}
}
