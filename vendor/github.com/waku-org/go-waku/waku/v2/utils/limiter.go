package utils

import (
	"context"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	sync.Mutex
	limiters *ttlcache.Cache[peer.ID, *rate.Limiter]
	r        rate.Limit
	b        int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		r: r,
		b: b,
		limiters: ttlcache.New[peer.ID, *rate.Limiter](
			ttlcache.WithTTL[peer.ID, *rate.Limiter](30 * time.Minute),
		),
	}
}

func (r *RateLimiter) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				r.Lock()
				r.limiters.DeleteExpired()
				r.Unlock()
			}
		}
	}()
}

func (r *RateLimiter) getOrCreate(peerID peer.ID) *rate.Limiter {
	r.Lock()
	defer r.Unlock()

	var limiter *rate.Limiter
	if !r.limiters.Has(peerID) {
		limiter = rate.NewLimiter(r.r, r.b)
		r.limiters.Set(peerID, limiter, ttlcache.DefaultTTL)
	} else {
		v := r.limiters.Get(peerID)
		limiter = v.Value()
	}
	return limiter
}

func (r *RateLimiter) Allow(peerID peer.ID) bool {
	return r.getOrCreate(peerID).Allow()
}

func (r *RateLimiter) Wait(ctx context.Context, peerID peer.ID) error {
	return r.getOrCreate(peerID).Wait(ctx)
}
