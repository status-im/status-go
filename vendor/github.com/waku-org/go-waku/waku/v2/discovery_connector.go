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
	cache       *lru.TwoQueueCache
	host        host.Host
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	minPeers    int
	dialTimeout time.Duration
	peerCh      chan peer.AddrInfo
	backoff     backoff.BackoffFactory
	mux         sync.Mutex
	logger      *zap.Logger
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
		peerCh:      make(chan peer.AddrInfo),
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

	c.wg.Add(1)
	go c.dialPeers(ctx)

	return nil
}

func (c *PeerConnectionStrategy) Stop() {
	if c.cancel == nil {
		return
	}

	c.cancel()
	c.wg.Wait()
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
		case pi, ok := <-c.peerCh:
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
			go func(pi peer.AddrInfo) {
				ctx, cancel := context.WithTimeout(ctx, c.dialTimeout)
				defer cancel()

				err := c.host.Connect(ctx, pi)
				if err != nil {
					c.logger.Info("connecting to peer", logging.HostID("peerID", pi.ID), zap.Error(err))
				}
				<-sem
			}(pi)

			ticker := time.NewTicker(1 * time.Second)
		peerCntLoop:
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if len(c.host.Network().Peers()) < c.minPeers {
						ticker.Stop()
						break peerCntLoop
					}
				}
			}
		case <-ctx.Done():
			if ctx.Err() != nil {
				c.logger.Info("discovery: backoff connector context error", zap.Error(ctx.Err()))
			}
			return
		}
	}
}
