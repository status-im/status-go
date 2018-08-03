package peers2

import (
	"sync"
	"time"

	"github.com/status-im/status-go/peers"
	"github.com/status-im/status-go/signal"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/discovery"
)

// DiscoveryContainer is an utility structure that wrapps
// discovery related objects. It provides an interface to
// control the whole discovery system.
type DiscoveryContainer struct {
	sync.RWMutex

	discovery discovery.Discovery
	peerPool  *PeerPool
	topics    []TopicPool

	wg   sync.WaitGroup
	quit chan struct{}
}

// NewDiscoveryContainer returns a new DiscoveryContainer instance.
func NewDiscoveryContainer(d discovery.Discovery, topics []TopicPool, cache *peers.Cache) *DiscoveryContainer {
	return &DiscoveryContainer{
		discovery: d,
		peerPool:  NewPeerPool(topics, cache),
	}
}

// Start starts all discovery related structures.
// If timeout is larger than 0, the system times out
// after this duration.
func (c *DiscoveryContainer) Start(server *p2p.Server, timeout time.Duration) error {
	if c.quit != nil {
		return nil
	}

	if err := c.startDiscovery(); err != nil {
		return err
	}
	signal.SendDiscoveryStarted()

	c.peerPool.Start(server)

	c.quit = make(chan struct{})

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
	c.Lock()
	defer c.Unlock()

	if c.quit == nil {
		return nil
	}

	close(c.quit)
	c.wg.Wait()
	c.quit = nil

	c.peerPool.Stop()
	signal.SendDiscoveryStopped()
	return c.stopDiscovery()
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

func (c *DiscoveryContainer) handleDiscoveryTimeout(timeout time.Duration, quit <-chan struct{}) {
	c.wg.Add(1)
	defer c.wg.Done()

	select {
	case <-quit:
	case <-time.After(timeout):
		go func() {
			if err := c.Stop(); err != nil {
				log.Error("failed to stop discovery", "err", err)
			}
		}()
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
			if !IsAllTopicsSatisfied(c.peerPool.Topics()) {
				continue
			}
			go func() {
				if err := c.Stop(); err != nil {
					log.Error("failed to stop discovery", "err", err)
				}
			}()
		}
	}
}
