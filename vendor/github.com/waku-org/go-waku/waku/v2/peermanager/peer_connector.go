package peermanager

// Adapted from github.com/libp2p/go-libp2p@v0.23.2/p2p/discovery/backoff/backoffconnector.go

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	"github.com/waku-org/go-waku/logging"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"

	"go.uber.org/zap"

	lru "github.com/hashicorp/golang-lru"
)

// PeerData contains information about a peer useful in establishing connections with it.
type PeerData struct {
	Origin   wps.Origin
	AddrInfo peer.AddrInfo
	ENR      *enode.Node
}

// PeerConnectionStrategy is a utility to connect to peers,
// but only if we have not recently tried connecting to them already
type PeerConnectionStrategy struct {
	sync.RWMutex

	cache  *lru.TwoQueueCache
	host   host.Host
	pm     *PeerManager
	cancel context.CancelFunc

	paused       bool
	workerCtx    context.Context
	workerCancel context.CancelFunc

	wg            sync.WaitGroup
	maxOutPeers   int
	dialTimeout   time.Duration
	peerCh        chan PeerData
	dialCh        chan peer.AddrInfo
	subscriptions []<-chan PeerData

	backoff backoff.BackoffFactory
	mux     sync.Mutex
	logger  *zap.Logger
}

// NewPeerConnectionStrategy creates a utility to connect to peers,
// but only if we have not recently tried connecting to them already.
//
// cacheSize is the size of a TwoQueueCache
// dialTimeout is how long we attempt to connect to a peer before giving up
// minPeers is the minimum number of peers that the node should have
// backoff describes the strategy used to decide how long to backoff after previously attempting to connect to a peer
func NewPeerConnectionStrategy(cacheSize int, maxOutPeers int,
	dialTimeout time.Duration, backoff backoff.BackoffFactory,
	logger *zap.Logger) (*PeerConnectionStrategy, error) {

	cache, err := lru.New2Q(cacheSize)
	if err != nil {
		return nil, err
	}

	return &PeerConnectionStrategy{
		cache:       cache,
		wg:          sync.WaitGroup{},
		maxOutPeers: maxOutPeers,
		dialTimeout: dialTimeout,
		backoff:     backoff,
		logger:      logger.Named("discovery-connector"),
	}, nil
}

type connCacheData struct {
	nextTry time.Time
	strat   backoff.BackoffStrategy
}

// Subscribe receives channels on which discovered peers should be pushed
func (c *PeerConnectionStrategy) Subscribe(ctx context.Context, ch <-chan PeerData) {
	if c.cancel != nil {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.consumeSubscription(ctx, ch)
		}()
	} else {
		c.subscriptions = append(c.subscriptions, ch)
	}
}

func (c *PeerConnectionStrategy) consumeSubscription(ctx context.Context, ch <-chan PeerData) {
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-ch:
			select {
			case <-ctx.Done():
				return
			case c.peerCh <- p:
			}
		}
	}

}

// SetHost sets the host to be able to mount or consume a protocol
func (c *PeerConnectionStrategy) SetHost(h host.Host) {
	c.host = h
}

// SetPeerManager sets the peermanager in order to utilize add peer
func (c *PeerConnectionStrategy) SetPeerManager(pm *PeerManager) {
	c.pm = pm
}

// Start attempts to connect to the peers passed in by peerCh.
// Will not connect to peers if they are within the backoff period.
func (c *PeerConnectionStrategy) Start(ctx context.Context) error {
	if c.cancel != nil {
		return errors.New("already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.peerCh = make(chan PeerData)
	c.dialCh = make(chan peer.AddrInfo)

	c.wg.Add(3)
	go c.shouldDialPeers(ctx)
	go c.workPublisher(ctx)
	go c.dialPeers(ctx)

	c.consumeSubscriptions(ctx)

	return nil
}

// Stop terminates the peer-connector
func (c *PeerConnectionStrategy) Stop() {
	if c.cancel == nil {
		return
	}

	c.cancel()
	c.wg.Wait()

	close(c.peerCh)
	close(c.dialCh)

	c.subscriptions = nil
	c.cancel = nil
}

func (c *PeerConnectionStrategy) isPaused() bool {
	c.RLock()
	defer c.RUnlock()
	return c.paused
}

func (c *PeerConnectionStrategy) shouldDialPeers(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	c.Lock()
	c.workerCtx, c.workerCancel = context.WithCancel(ctx)
	c.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			isPaused := c.isPaused()
			_, outRelayPeers, err := c.pm.GroupPeersByDirection()
			if err != nil {
				c.logger.Warn("failed to get outRelayPeers from peerstore", zap.Error(err))
				continue
			}
			numPeers := outRelayPeers.Len()
			if numPeers >= c.maxOutPeers && !isPaused {
				c.Lock()
				c.paused = true
				c.workerCancel()
				c.Unlock()
			} else if numPeers < c.maxOutPeers && isPaused {
				c.Lock()
				c.paused = false
				c.workerCtx, c.workerCancel = context.WithCancel(ctx)
				c.Unlock()
			}
		}
	}
}

func (c *PeerConnectionStrategy) consumeSubscriptions(ctx context.Context) {
	for _, subs := range c.subscriptions {
		c.wg.Add(1)
		go func(s <-chan PeerData) {
			defer c.wg.Done()
			c.consumeSubscription(ctx, s)
		}(subs)
	}
}

func (c *PeerConnectionStrategy) publishWork(ctx context.Context, p peer.AddrInfo) {
	select {
	case c.dialCh <- p:
	case <-ctx.Done():
		return
	}
}

func (c *PeerConnectionStrategy) workPublisher(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			isPaused := c.isPaused()
			if !isPaused {
				select {
				case <-ctx.Done():
					return
				case p := <-c.peerCh:
					c.pm.AddDiscoveredPeer(p)
					c.publishWork(ctx, p.AddrInfo)
				case <-time.After(1 * time.Second):
					// This timeout is to not lock the goroutine
					break
				}
			} else {
				// Check if paused again
				time.Sleep(1 * time.Second)
			}
		}
	}
}

const maxActiveDials = 5

func (c *PeerConnectionStrategy) canDialPeer(pi peer.AddrInfo) bool {
	c.mux.Lock()
	val, ok := c.cache.Get(pi.ID)
	var cachedPeer *connCacheData
	if ok {
		tv := val.(*connCacheData)
		now := time.Now()
		if now.Before(tv.nextTry) {
			c.mux.Unlock()
			return false
		}

		tv.nextTry = now.Add(tv.strat.Delay())
	} else {
		cachedPeer = &connCacheData{strat: c.backoff()}
		cachedPeer.nextTry = time.Now().Add(cachedPeer.strat.Delay())
		c.cache.Add(pi.ID, cachedPeer)
	}
	c.mux.Unlock()
	return true
}

func (c *PeerConnectionStrategy) dialPeers(ctx context.Context) {
	defer c.wg.Done()

	maxGoRoutines := c.maxOutPeers
	if maxGoRoutines > maxActiveDials {
		maxGoRoutines = maxActiveDials
	}

	sem := make(chan struct{}, maxGoRoutines)

	for {
		select {
		case pi, ok := <-c.dialCh:
			if !ok {
				return
			}

			if pi.ID == c.host.ID() || pi.ID == "" ||
				c.host.Network().Connectedness(pi.ID) == network.Connected {
				continue
			}

			if c.canDialPeer(pi) {
				sem <- struct{}{}
				c.wg.Add(1)
				go c.dialPeer(pi, sem)
			} else {
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *PeerConnectionStrategy) dialPeer(pi peer.AddrInfo, sem chan struct{}) {
	defer c.wg.Done()
	c.RLock()
	ctx, cancel := context.WithTimeout(c.workerCtx, c.dialTimeout)
	c.RUnlock()
	defer cancel()
	err := c.host.Connect(ctx, pi)
	if err != nil && !errors.Is(err, context.Canceled) {
		c.host.Peerstore().(wps.WakuPeerstore).AddConnFailure(pi)
		c.logger.Warn("connecting to peer", logging.HostID("peerID", pi.ID), zap.Error(err))
	}
	<-sem
}
