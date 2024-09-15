package market

import (
	"sync"
)

type CacheStore[T any] interface {
	CreateStore() T
}

type Cache[T CacheStore[T]] struct {
	store T
	lock  sync.RWMutex
}

func NewCache[T CacheStore[T]]() *Cache[T] {
	var cache Cache[T]
	cache.store = cache.store.CreateStore()
	return &cache
}

func Read[T CacheStore[T], R any](cache *Cache[T], reader func(store T) R) R {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	return reader(cache.store)
}

func Write[T CacheStore[T]](cache *Cache[T], writer func(store *T) *T) *Cache[T] {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.store = *writer(&cache.store)
	return cache
}

func (cache *Cache[T]) Get() T {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	return cache.store
}

func (cache *Cache[T]) Set(update func(T) T) *Cache[T] {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.store = update(cache.store)
	return cache
}
