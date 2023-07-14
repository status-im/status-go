package v2

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
	"github.com/libp2p/go-libp2p/core/peerstore"

	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/peers"

	"go.uber.org/zap"

	lru "github.com/hashicorp/golang-lru"
)

// PeerConnectionStrategy is a utility to connect to peers, but only if we have not recently tried connecting to them already
type PeerConnectionStrategy struct {
	sync.RWMutex

	cache  *lru.TwoQueueCache
	host   host.Host
	cancel context.CancelFunc

	paused       bool
	workerCtx    context.Context
	workerCancel context.CancelFunc

	wg            sync.WaitGroup
	minPeers      int
	dialTimeout   time.Duration
	peerCh        chan PeerData
	dialCh        chan peer.AddrInfo
	subscriptions []<-chan PeerData

	backoff backoff.BackoffFactory
	mux     sync.Mutex
	logger  *zap.Logger
}

// NewPeerConnectionStrategy creates a utility to connect to peers, but only if we have not recently tried connecting to them already.
// cacheSize is the size of a TwoQueueCache
// dialTimeout is how long we attempt to connect to a peer before giving up
// minPeers is the minimum number of peers that the node should have
// backoff describes the strategy used to decide how long to backoff after previously attempting to connect to a peer
func NewPeerConnectionStrategy(cacheSize int, minPeers int, dialTimeout time.Duration, backoff backoff.BackoffFactory, logger *zap.Logger) (*PeerConnectionStrategy, error) {
	cache, err := lru.New2Q(cacheSize)
	if err != nil {
		return nil, err
	}

	return &PeerConnectionStrategy{
		cache:       cache,
		wg:          sync.WaitGroup{},
		minPeers:    minPeers,
		dialTimeout: dialTimeout,
		backoff:     backoff,
		logger:      logger.Named("discovery-connector"),
	}, nil
}

type connCacheData struct {
	nextTry time.Time
	strat   backoff.BackoffStrategy
}

type PeerData struct {
	Origin   peers.Origin
	AddrInfo peer.AddrInfo
	ENR      *enode.Node
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

// Sets the host to be able to mount or consume a protocol
func (c *PeerConnectionStrategy) SetHost(h host.Host) {
	c.host = h
}

// Start attempts to connect to the peers passed in by peerCh. Will not connect to peers if they are within the backoff period.
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
			numPeers := len(c.host.Network().Peers())
			if numPeers >= c.minPeers && !isPaused {
				c.Lock()
				c.paused = true
				c.workerCancel()
				c.Unlock()
			} else if numPeers < c.minPeers && isPaused {
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
					c.host.Peerstore().AddAddrs(p.AddrInfo.ID, p.AddrInfo.Addrs, peerstore.AddressTTL)
					err := c.host.Peerstore().(peers.WakuPeerstore).SetOrigin(p.AddrInfo.ID, p.Origin)
					if err != nil {
						c.logger.Error("could not set origin", zap.Error(err), logging.HostID("peer", p.AddrInfo.ID))
					}

					if p.ENR != nil {
						err = c.host.Peerstore().(peers.WakuPeerstore).SetENR(p.AddrInfo.ID, p.ENR)
						if err != nil {
							c.logger.Error("could not store enr", zap.Error(err), logging.HostID("peer", p.AddrInfo.ID), zap.String("enr", p.ENR.String()))
						}
					}

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

func (c *PeerConnectionStrategy) dialPeers(ctx context.Context) {
	defer c.wg.Done()
	for {
		select {
		case pi, ok := <-c.dialCh:
			if !ok {
				return
			}

			if pi.ID == c.host.ID() || pi.ID == "" {
				continue
			}

			if c.host.Network().Connectedness(pi.ID) == network.Connected {
				continue
			}

			c.mux.Lock()
			val, ok := c.cache.Get(pi.ID)
			var cachedPeer *connCacheData
			if ok {
				tv := val.(*connCacheData)
				now := time.Now()
				if now.Before(tv.nextTry) {
					c.mux.Unlock()
					continue
				}

				tv.nextTry = now.Add(tv.strat.Delay())
			} else {
				cachedPeer = &connCacheData{strat: c.backoff()}
				cachedPeer.nextTry = time.Now().Add(cachedPeer.strat.Delay())
				c.cache.Add(pi.ID, cachedPeer)
			}
			c.mux.Unlock()

			dialCtx, dialCtxCancel := context.WithTimeout(c.workerCtx, c.dialTimeout)
			err := c.host.Connect(dialCtx, pi)
			if err != nil && !errors.Is(err, context.Canceled) {
				c.host.Peerstore().(peers.WakuPeerstore).AddConnFailure(pi)
				c.logger.Info("connecting to peer", logging.HostID("peerID", pi.ID), zap.Error(err))
			}
			dialCtxCancel()

		case <-ctx.Done():
			return
		}
	}
}
