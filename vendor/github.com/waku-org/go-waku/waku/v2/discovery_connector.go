package v2

// Adapted from github.com/libp2p/go-libp2p@v0.23.2/p2p/discovery/backoff/backoffconnector.go

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	"github.com/waku-org/go-waku/logging"
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

	wg          sync.WaitGroup
	minPeers    int
	dialTimeout time.Duration
	peerCh      chan peer.AddrInfo
	dialCh      chan peer.AddrInfo

	backoff backoff.BackoffFactory
	mux     sync.Mutex
	logger  *zap.Logger
}

// NewPeerConnectionStrategy creates a utility to connect to peers, but only if we have not recently tried connecting to them already.
// cacheSize is the size of a TwoQueueCache
// dialTimeout is how long we attempt to connect to a peer before giving up
// minPeers is the minimum number of peers that the node should have
// backoff describes the strategy used to decide how long to backoff after previously attempting to connect to a peer
func NewPeerConnectionStrategy(h host.Host, cacheSize int, minPeers int, dialTimeout time.Duration, backoff backoff.BackoffFactory, logger *zap.Logger) (*PeerConnectionStrategy, error) {
	cache, err := lru.New2Q(cacheSize)
	if err != nil {
		return nil, err
	}

	return &PeerConnectionStrategy{
		cache:       cache,
		host:        h,
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

// PeerChannel exposes the channel on which discovered peers should be pushed
func (c *PeerConnectionStrategy) PeerChannel() chan<- peer.AddrInfo {
	return c.peerCh
}

// Start attempts to connect to the peers passed in by peerCh. Will not connect to peers if they are within the backoff period.
func (c *PeerConnectionStrategy) Start(ctx context.Context) error {
	if c.cancel != nil {
		return errors.New("already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.peerCh = make(chan peer.AddrInfo)
	c.dialCh = make(chan peer.AddrInfo)

	c.wg.Add(3)
	go c.shouldDialPeers(ctx)
	go c.workPublisher(ctx)
	go c.dialPeers(ctx)

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

func (c *PeerConnectionStrategy) publishWork(ctx context.Context, p peer.AddrInfo) {
	select {
	case c.dialCh <- p:
	case <-ctx.Done():
		return
	case <-time.After(1 * time.Second):
		// This timeout is to not lock the goroutine
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
					c.publishWork(ctx, p)
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

	maxGoRoutines := c.minPeers
	if maxGoRoutines > 15 {
		maxGoRoutines = 15
	}

	sem := make(chan struct{}, maxGoRoutines)
	for {
		select {
		case pi, ok := <-c.dialCh:
			if !ok {
				return
			}

			if pi.ID == c.host.ID() || pi.ID == "" {
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

			if c.host.Network().Connectedness(pi.ID) == network.Connected {
				continue
			}

			sem <- struct{}{}
			c.wg.Add(1)
			go func(pi peer.AddrInfo) {
				defer c.wg.Done()

				ctx, cancel := context.WithTimeout(c.workerCtx, c.dialTimeout)
				defer cancel()
				err := c.host.Connect(ctx, pi)
				if err != nil && !errors.Is(err, context.Canceled) {
					c.logger.Info("connecting to peer", logging.HostID("peerID", pi.ID), zap.Error(err))
				}
				<-sem
			}(pi)
		case <-ctx.Done():
			return
		}
	}
}
