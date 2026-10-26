package coingecko

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/status-im/status-go/internal/traffic"
)

const (
	// coinsListTTL is how long the coingecko id mapping is reused before it is revalidated.
	coinsListTTL = time.Hour
	// coinsListRetryDelay is how long a stale mapping is reused after a failed revalidation.
	coinsListRetryDelay = time.Minute
)

// coinsListCache keeps the token key to coingecko id mapping built from /coins/list.
// The zero value is an empty cache.
type coinsListCache struct {
	group singleflight.Group

	mu        sync.RWMutex
	byKey     map[string]string
	etag      string
	expiresAt time.Time

	now func() time.Time
}

func (cc *coinsListCache) clock() time.Time {
	if cc.now != nil {
		return cc.now()
	}
	return time.Now()
}

func (cc *coinsListCache) fresh() (map[string]string, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.byKey, cc.byKey != nil && cc.clock().Before(cc.expiresAt)
}

// getCoingeckoIDsByTokenKey returns a map of token keys to coingecko ids.
// The map is shared between callers and must not be modified.
func (c *Client) getCoingeckoIDsByTokenKey() (map[string]string, error) {
	if byKey, ok := c.coinsList.fresh(); ok {
		return byKey, nil
	}

	byKey, err, _ := c.coinsList.group.Do("coins-list", func() (interface{}, error) {
		if byKey, ok := c.coinsList.fresh(); ok {
			return byKey, nil
		}
		return c.refreshCoinsList()
	})
	if err != nil {
		return nil, err
	}
	return byKey.(map[string]string), nil
}

func (c *Client) refreshCoinsList() (map[string]string, error) {
	cc := &c.coinsList

	cc.mu.RLock()
	cached, etag := cc.byKey, cc.etag
	cc.mu.RUnlock()
	if cached == nil {
		etag = ""
	}

	ctx := traffic.WithSource(context.Background(), traffic.MarketTokenList)
	tokens, newEtag, notModified, err := c.fetchTokensWithEtag(ctx, etag)

	cc.mu.Lock()
	defer cc.mu.Unlock()
	now := cc.clock()
	switch {
	case err != nil:
		if cached == nil {
			return nil, err
		}
		cc.expiresAt = now.Add(coinsListRetryDelay)
	case notModified:
		cc.expiresAt = now.Add(coinsListTTL)
	default:
		cc.byKey = buildCoingeckoIDsByTokenKey(tokens)
		cc.etag = newEtag
		cc.expiresAt = now.Add(coinsListTTL)
	}
	return cc.byKey, nil
}
