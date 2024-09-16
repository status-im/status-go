package market

import (
	"sync"
)

type MarketCache[T any] struct {
	store T
	lock  sync.RWMutex
}

func NewCache[T any](store T) *MarketCache[T] {
	var cache MarketCache[T]
	cache.store = store
	return &cache
}

func Read[T any, R any](cache *MarketCache[T], reader func(store T) R) R {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	return reader(cache.store)
}

func Write[T any](cache *MarketCache[T], writer func(store T) T) *MarketCache[T] {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.store = writer(cache.store)
	return cache
}
