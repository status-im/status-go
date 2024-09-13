package market

import (
	"sync"
	"time"
)

type cacheItem[V any] struct {
	value      V
	expiration time.Time
}

type Cache[K comparable, V any] struct {
	data    map[K]cacheItem[V]
	lock    sync.RWMutex
	ttl     time.Duration
	fetcher func(key K) (V, error)
}

func NewCache[K comparable, V any](ttl time.Duration, fetchFn func(key K) (V, error)) *Cache[K, V] {
	return &Cache[K, V]{
		data:    make(map[K]cacheItem[V]),
		ttl:     ttl,
		fetcher: fetchFn,
	}
}

func (c *Cache[K, V]) Get(key K, fresh bool) (V, error) {
	if fresh {
		return c.refresh(key, fresh)
	}

	c.lock.RLock()
	item, exists := c.data[key]
	c.lock.RUnlock()

	if exists && time.Now().Before(item.expiration) {
		return item.value, nil
	}

	return c.refresh(key, fresh)
}

func (c *Cache[K, V]) refresh(key K, fresh bool) (V, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !fresh {
		item, exists := c.data[key]
		if exists && time.Now().Before(item.expiration) {
			return item.value, nil
		}
	}

	value, err := c.fetcher(key)
	if err != nil {
		var zero V
		return zero, err
	}

	c.data[key] = cacheItem[V]{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}

	return value, nil
}
