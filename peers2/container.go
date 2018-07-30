package peers2

import (
	"sync"
	"time"

	"github.com/status-im/status-go/peers"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/discovery"
)

type DiscoveryContainer struct {
	discovery discovery.Discovery
	peerPool  *PeerPool
	topics    []TopicPool

	discoveryRunning bool

	wg   sync.WaitGroup
	quit chan struct{}
}

func NewDiscoveryContainer(
	d discovery.Discovery, topics []TopicPool, cache *peers.Cache,
) *DiscoveryContainer {
	return &DiscoveryContainer{
		discovery: d,
		peerPool:  NewPeerPool(topics, cache),
		topics:    topics,
	}
}

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

	err = c.startDiscovery()
	if err != nil {
		return
	}

	c.quit = make(chan struct{})

	if timeout > 0 {
		go c.handleTimeout(time.After(timeout))
	}
	go c.checkTopicSatisfaction(time.Second)

	return nil
}

func (c *DiscoveryContainer) Stop() error {
	if c.quit == nil {
		return nil
	}

	close(c.quit)
	c.wg.Wait()

	return c.stopDiscovery()
}

func (c *DiscoveryContainer) startDiscovery() error {
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

func (c *DiscoveryContainer) stopDiscovery() (err error) {
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

func (c *DiscoveryContainer) checkTopicSatisfaction(period time.Duration) {
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
				c.stopDiscovery()
			} else {
				log.Debug("not all topics are satisfied")
				c.startDiscovery()
			}
		}
	}
}
