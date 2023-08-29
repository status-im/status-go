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

	"sync/atomic"

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

	paused atomic.Bool

	wg            sync.WaitGroup
	dialTimeout   time.Duration
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
func NewPeerConnectionStrategy(cacheSize int, pm *PeerManager,
	dialTimeout time.Duration, backoff backoff.BackoffFactory,
	logger *zap.Logger) (*PeerConnectionStrategy, error) {

	cache, err := lru.New2Q(cacheSize)
	if err != nil {
		return nil, err
	}
	pc := &PeerConnectionStrategy{
		cache:       cache,
		wg:          sync.WaitGroup{},
		dialTimeout: dialTimeout,
		pm:          pm,
		backoff:     backoff,
		logger:      logger.Named("discovery-connector"),
	}
	pm.SetPeerConnector(pc)
	return pc, nil
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
		// for returning from the loop when peerConnector is paused.
		select {
		case <-ctx.Done():
			return
		default:
		}
		//
		if !c.isPaused() {
			select {
			case <-ctx.Done():
				return
			case p, ok := <-ch:
				if !ok {
					return
				}
				c.pm.AddDiscoveredPeer(p)
				c.publishWork(ctx, p.AddrInfo)
			case <-time.After(1 * time.Second):
				// This timeout is to not lock the goroutine
				break
			}
		} else {
			time.Sleep(1 * time.Second) // sleep while the peerConnector is paused.
		}
	}
}

// SetHost sets the host to be able to mount or consume a protocol
func (c *PeerConnectionStrategy) SetHost(h host.Host) {
	c.host = h
}

// Start attempts to connect to the peers passed in by peerCh.
// Will not connect to peers if they are within the backoff period.
func (c *PeerConnectionStrategy) Start(ctx context.Context) error {
	if c.cancel != nil {
		return errors.New("already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.dialCh = make(chan peer.AddrInfo)

	c.wg.Add(2)
	go c.shouldDialPeers(ctx)
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
	c.cancel = nil
	c.wg.Wait()

	close(c.dialCh)
}

func (c *PeerConnectionStrategy) isPaused() bool {
	return c.paused.Load()
}

func (c *PeerConnectionStrategy) shouldDialPeers(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, outRelayPeers := c.pm.getRelayPeers()
			c.paused.Store(outRelayPeers.Len() >= c.pm.OutRelayPeersTarget) // pause if no of OutPeers more than or eq to target
		}
	}
}

// it might happen Subscribe is called before peerConnector has started so store these subscriptions in subscriptions array and custom after c.cancel is set.
func (c *PeerConnectionStrategy) consumeSubscriptions(ctx context.Context) {
	for _, subs := range c.subscriptions {
		c.wg.Add(1)
		go func(s <-chan PeerData) {
			defer c.wg.Done()
			c.consumeSubscription(ctx, s)
		}(subs)
	}
	c.subscriptions = nil
}

func (c *PeerConnectionStrategy) publishWork(ctx context.Context, p peer.AddrInfo) {
	select {
	case c.dialCh <- p:
	case <-ctx.Done():
		return
	}
}

const maxActiveDials = 5

// c.cache is thread safe
// only reason why mutex is used: if canDialPeer is queried twice for the same peer.
func (c *PeerConnectionStrategy) canDialPeer(pi peer.AddrInfo) bool {
	c.mux.Lock()
	defer c.mux.Unlock()
	val, ok := c.cache.Get(pi.ID)
	var cachedPeer *connCacheData
	if ok {
		tv := val.(*connCacheData)
		now := time.Now()
		if now.Before(tv.nextTry) {
			return false
		}

		tv.nextTry = now.Add(tv.strat.Delay())
	} else {
		cachedPeer = &connCacheData{strat: c.backoff()}
		cachedPeer.nextTry = time.Now().Add(cachedPeer.strat.Delay())
		c.cache.Add(pi.ID, cachedPeer)
	}
	return true
}

func (c *PeerConnectionStrategy) dialPeers(ctx context.Context) {
	defer c.wg.Done()

	maxGoRoutines := c.pm.OutRelayPeersTarget
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
				go c.dialPeer(ctx, pi, sem)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *PeerConnectionStrategy) dialPeer(ctx context.Context, pi peer.AddrInfo, sem chan struct{}) {
	defer c.wg.Done()
	ctx, cancel := context.WithTimeout(ctx, c.dialTimeout)
	defer cancel()
	err := c.host.Connect(ctx, pi)
	if err != nil && !errors.Is(err, context.Canceled) {
		c.host.Peerstore().(wps.WakuPeerstore).AddConnFailure(pi)
		c.logger.Warn("connecting to peer", logging.HostID("peerID", pi.ID), zap.Error(err))
	}
	<-sem
}
