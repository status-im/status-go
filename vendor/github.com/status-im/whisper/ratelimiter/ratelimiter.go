package ratelimiter

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/juju/ratelimit"
	"github.com/syndtr/goleveldb/leveldb"
)

// Interface describes common ratelimiter methods.
type Interface interface {
	Create([]byte) error
	Remove([]byte, time.Duration) error
	TakeAvailable([]byte, int64) int64
	Available([]byte) int64
	UpdateConfig([]byte, Config) error
	Config() Config
}

// Config is a set of options used by rate limiter.
type Config struct {
	Interval, Capacity, Quantum uint64
}

// compare config with existing ratelimited bucket.
func compare(c Config, bucket *ratelimit.Bucket) bool {
	return int64(c.Capacity) == bucket.Capacity() &&
		1e9*float64(c.Quantum)/float64(c.Interval) == bucket.Rate()
}

func newBucket(c Config) *ratelimit.Bucket {
	return ratelimit.NewBucketWithQuantum(time.Duration(c.Interval), int64(c.Capacity), int64(c.Quantum))
}

func NewPersisted(db DBInterface, config Config) *PersistedRateLimiter {
	return &PersistedRateLimiter{
		db:            db,
		defaultConfig: config,
		initialized:   map[string]*ratelimit.Bucket{},
		timeFunc:      time.Now,
	}
}

// PersistedRateLimiter persists latest capacity and updated config per unique ID.
type PersistedRateLimiter struct {
	db            DBInterface
	defaultConfig Config

	mu          sync.Mutex
	initialized map[string]*ratelimit.Bucket

	timeFunc func() time.Time
}

func (r *PersistedRateLimiter) blacklist(id []byte, duration time.Duration) error {
	if duration == 0 {
		return nil
	}
	record := BlacklistRecord{ID: id, Deadline: r.timeFunc().Add(duration)}
	if err := record.Write(r.db); err != nil {
		return fmt.Errorf("error blacklisting %x: %v", id, err)
	}
	return nil
}

func (r *PersistedRateLimiter) Config() Config {
	return r.defaultConfig
}

func (r *PersistedRateLimiter) getOrCreate(id []byte, config Config) (bucket *ratelimit.Bucket) {
	r.mu.Lock()
	defer r.mu.Unlock()
	old, exist := r.initialized[string(id)]
	if !exist {
		bucket = newBucket(config)
		r.initialized[string(id)] = bucket
	} else {
		bucket = old
	}
	return
}

func (r *PersistedRateLimiter) Create(id []byte) error {
	bl := BlacklistRecord{ID: id}
	if err := bl.Read(r.db); err != leveldb.ErrNotFound {
		if bl.Deadline.After(r.timeFunc()) {
			return fmt.Errorf("identity %x is blacklisted", id)
		}
		bl.Remove(r.db)
	}
	bucket := r.getOrCreate(id, r.defaultConfig)
	capacity := CapacityRecord{ID: id}
	if err := capacity.Read(r.db); err != nil {
		return nil
	}
	bucket.TakeAvailable(capacity.Taken)
	// TODO refill rate limiter due to time difference. e.g. if record was stored at T and C seconds passed since T.
	// we need to add RATE_PER_SECOND*C to a bucket
	return nil
}

// Remove removes key from memory but ensures that the latest information is persisted.
func (r *PersistedRateLimiter) Remove(id []byte, duration time.Duration) error {
	if duration != 0 {
		if err := r.blacklist(id, duration); err != nil {
			return err
		}
	}
	r.mu.Lock()
	bucket, exist := r.initialized[string(id)]
	delete(r.initialized, string(id))
	r.mu.Unlock()
	if !exist || bucket == nil {
		return nil
	}
	return r.store(id, bucket)
}

func (r *PersistedRateLimiter) store(id []byte, bucket *ratelimit.Bucket) error {
	capacity := CapacityRecord{
		ID:        id,
		Taken:     bucket.Capacity() - bucket.Available(),
		Timestamp: r.timeFunc(),
	}
	if err := capacity.Write(r.db); err != nil {
		return fmt.Errorf("failed to write current capacicity %d for id %x: %v",
			bucket.Capacity(), id, err)
	}
	return nil
}

func (r *PersistedRateLimiter) TakeAvailable(id []byte, count int64) int64 {
	bucket := r.getOrCreate(id, r.defaultConfig)
	rst := bucket.TakeAvailable(count)
	if err := r.store(id, bucket); err != nil {
		log.Error(err.Error())
	}
	return rst
}

func (r *PersistedRateLimiter) Available(id []byte) int64 {
	return r.getOrCreate(id, r.defaultConfig).Available()
}

func (r *PersistedRateLimiter) UpdateConfig(id []byte, config Config) error {
	r.mu.Lock()
	old, _ := r.initialized[string(id)]
	if compare(config, old) {
		r.mu.Unlock()
		return nil
	}
	delete(r.initialized, string(id))
	r.mu.Unlock()
	taken := int64(0)
	if old != nil {
		taken = old.Capacity() - old.Available()
	}
	r.getOrCreate(id, config).TakeAvailable(taken)
	return nil
}
