package market

import (
	"sync"
)

type CacheStore[T any] interface {
	CreateStore() T
}

type MarketCache[T CacheStore[T]] struct {
	store T
	lock  sync.RWMutex
}

func NewCache[T CacheStore[T]]() *MarketCache[T] {
	var cache MarketCache[T]
	cache.store = cache.store.CreateStore()
	return &cache
}

func Read[T CacheStore[T], R any](cache *MarketCache[T], reader func(store T) R) R {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	return reader(cache.store)
}

func Write[T CacheStore[T]](cache *MarketCache[T], writer func(store *T) *T) *MarketCache[T] {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.store = *writer(&cache.store)
	return cache
}

func (cache *MarketCache[T]) Get() T {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	return cache.store
}

func (cache *MarketCache[T]) Set(update func(T) T) *MarketCache[T] {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.store = update(cache.store)
	return cache
}
