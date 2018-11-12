package peerstore

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
)

// Number of times to retry transactional writes
var dsWriteRetries = 5

// DatastoreAddrManager is an address manager backed by a Datastore with both an
// in-memory TTL manager and an in-memory address stream manager.
type DatastoreAddrManager struct {
	cache       *lru.ARCCache
	ds          ds.Batching
	ttlManager  *ttlmanager
	subsManager *AddrSubManager
}

// NewDatastoreAddrManager initializes a new DatastoreAddrManager given a
// Datastore instance, a context for managing the TTL manager, and the interval
// at which the TTL manager should sweep the Datastore.
func NewDatastoreAddrManager(ctx context.Context, ds ds.Batching, ttlInterval time.Duration) (*DatastoreAddrManager, error) {
	cache, err := lru.NewARC(1024)
	if err != nil {
		return nil, err
	}

	mgr := &DatastoreAddrManager{
		cache:       cache,
		ds:          ds,
		ttlManager:  newTTLManager(ctx, ds, cache, ttlInterval),
		subsManager: NewAddrSubManager(),
	}
	return mgr, nil
}

// Stop will signal the TTL manager to stop and block until it returns.
func (mgr *DatastoreAddrManager) Stop() {
	mgr.ttlManager.cancel()
}

func peerAddressKey(p *peer.ID, addr *ma.Multiaddr) (ds.Key, error) {
	hash, err := mh.Sum((*addr).Bytes(), mh.MURMUR3, -1)
	if err != nil {
		return ds.Key{}, nil
	}
	return ds.NewKey(peer.IDB58Encode(*p)).ChildString(hash.B58String()), nil
}

func peerIDFromKey(key ds.Key) (peer.ID, error) {
	idstring := key.Parent().Name()
	return peer.IDB58Decode(idstring)
}

// AddAddr will add a new address if it's not already in the AddrBook.
func (mgr *DatastoreAddrManager) AddAddr(p peer.ID, addr ma.Multiaddr, ttl time.Duration) {
	mgr.AddAddrs(p, []ma.Multiaddr{addr}, ttl)
}

// AddAddrs will add many new addresses if they're not already in the AddrBook.
func (mgr *DatastoreAddrManager) AddAddrs(p peer.ID, addrs []ma.Multiaddr, ttl time.Duration) {
	if ttl <= 0 {
		return
	}

	mgr.setAddrs(p, addrs, ttl, true)
}

// SetAddr will add or update the TTL of an address in the AddrBook.
func (mgr *DatastoreAddrManager) SetAddr(p peer.ID, addr ma.Multiaddr, ttl time.Duration) {
	mgr.SetAddrs(p, []ma.Multiaddr{addr}, ttl)
}

// SetAddrs will add or update the TTLs of addresses in the AddrBook.
func (mgr *DatastoreAddrManager) SetAddrs(p peer.ID, addrs []ma.Multiaddr, ttl time.Duration) {
	mgr.setAddrs(p, addrs, ttl, false)
}

func (mgr *DatastoreAddrManager) setAddrs(p peer.ID, addrs []ma.Multiaddr, ttl time.Duration, add bool) {
	for i := 0; i < dsWriteRetries; i++ {
		// keys to add to the TTL manager
		var keys []ds.Key
		batch, err := mgr.ds.Batch()
		if err != nil {
			log.Error(err)
			return
		}

		for _, addr := range addrs {
			if addr == nil {
				continue
			}

			key, err := peerAddressKey(&p, &addr)
			if err != nil {
				log.Error(err)
				continue
			}
			keys = append(keys, key)

			if ttl <= 0 {
				if err := batch.Delete(key); err != nil {
					log.Error(err)
				} else {
					mgr.cache.Remove(key)
				}
				continue
			}

			has := mgr.cache.Contains(key)
			if !has {
				has, err = mgr.ds.Has(key)
			}
			if err != nil || !has {
				mgr.subsManager.BroadcastAddr(p, addr)
			}

			// Allows us to support AddAddr and SetAddr in one function
			if !has {
				if err := batch.Put(key, addr.Bytes()); err != nil {
					log.Error(err)
				} else {
					mgr.cache.Add(key, addr.Bytes())
				}
			}
		}
		if err := batch.Commit(); err != nil {
			log.Errorf("failed to write addresses for peer %s: %s\n", p.Pretty(), err)
			continue
		}
		mgr.ttlManager.setTTLs(keys, ttl, add)
		return
	}
	log.Errorf("failed to avoid write conflict for peer %s after %d retries\n", p.Pretty(), dsWriteRetries)
}

// UpdateAddrs will update any addresses for a given peer and TTL combination to
// have a new TTL.
func (mgr *DatastoreAddrManager) UpdateAddrs(p peer.ID, oldTTL time.Duration, newTTL time.Duration) {
	prefix := ds.NewKey(p.Pretty())
	mgr.ttlManager.updateTTLs(prefix, oldTTL, newTTL)
}

// Addrs Returns all of the non-expired addresses for a given peer.
func (mgr *DatastoreAddrManager) Addrs(p peer.ID) []ma.Multiaddr {
	prefix := ds.NewKey(p.Pretty())
	q := query.Query{Prefix: prefix.String(), KeysOnly: true}
	results, err := mgr.ds.Query(q)
	if err != nil {
		log.Error(err)
		return nil
	}

	var addrs []ma.Multiaddr
	for result := range results.Next() {
		key := ds.RawKey(result.Key)
		var addri interface{}
		addri, ok := mgr.cache.Get(key)
		if !ok {
			addri, err = mgr.ds.Get(key)
			if err != nil {
				log.Error(err)
				continue
			}
		}
		addrbytes := addri.([]byte)
		addr, err := ma.NewMultiaddrBytes(addrbytes)
		if err != nil {
			log.Error(err)
			continue
		}
		addrs = append(addrs, addr)
	}

	return addrs
}

// Peers returns all of the peer IDs for which the AddrBook has addresses.
func (mgr *DatastoreAddrManager) Peers() []peer.ID {
	q := query.Query{KeysOnly: true}
	results, err := mgr.ds.Query(q)
	if err != nil {
		log.Error(err)
		return []peer.ID{}
	}

	idset := make(map[peer.ID]struct{})
	for result := range results.Next() {
		key := ds.RawKey(result.Key)
		id, err := peerIDFromKey(key)
		if err != nil {
			continue
		}
		idset[id] = struct{}{}
	}

	ids := make([]peer.ID, 0, len(idset))
	for id := range idset {
		ids = append(ids, id)
	}
	return ids
}

// AddrStream returns a channel on which all new addresses discovered for a
// given peer ID will be published.
func (mgr *DatastoreAddrManager) AddrStream(ctx context.Context, p peer.ID) <-chan ma.Multiaddr {
	initial := mgr.Addrs(p)
	return mgr.subsManager.AddrStream(ctx, p, initial)
}

// ClearAddrs will delete all known addresses for a peer ID.
func (mgr *DatastoreAddrManager) ClearAddrs(p peer.ID) {
	prefix := ds.NewKey(p.Pretty())
	for i := 0; i < dsWriteRetries; i++ {
		q := query.Query{Prefix: prefix.String(), KeysOnly: true}
		results, err := mgr.ds.Query(q)
		if err != nil {
			log.Error(err)
			return
		}
		batch, err := mgr.ds.Batch()
		if err != nil {
			log.Error(err)
			return
		}

		for result := range results.Next() {
			key := ds.NewKey(result.Key)
			err := batch.Delete(key)
			if err != nil {
				// From inspectin badger, errors here signify a problem with
				// the transaction as a whole, so we can log and abort.
				log.Error(err)
				return
			}
			mgr.cache.Remove(key)
		}
		if err = batch.Commit(); err != nil {
			log.Errorf("failed to clear addresses for peer %s: %s\n", p.Pretty(), err)
			continue
		}
		mgr.ttlManager.clear(ds.NewKey(p.Pretty()))
		return
	}
	log.Errorf("failed to clear addresses for peer %s after %d attempts\n", p.Pretty(), dsWriteRetries)
}

// ttlmanager

type ttlentry struct {
	TTL       time.Duration
	ExpiresAt time.Time
}

type ttlmanager struct {
	sync.RWMutex
	entries map[ds.Key]*ttlentry

	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
	ds     ds.Batching
	cache  *lru.ARCCache
}

func newTTLManager(parent context.Context, d ds.Datastore, c *lru.ARCCache, tick time.Duration) *ttlmanager {
	ctx, cancel := context.WithCancel(parent)
	batching, ok := d.(ds.Batching)
	if !ok {
		panic("must construct ttlmanager with batching datastore")
	}
	mgr := &ttlmanager{
		entries: make(map[ds.Key]*ttlentry),
		ctx:     ctx,
		cancel:  cancel,
		ticker:  time.NewTicker(tick),
		ds:      batching,
		cache:   c,
	}

	go func() {
		for {
			select {
			case <-mgr.ctx.Done():
				mgr.ticker.Stop()
				return
			case <-mgr.ticker.C:
				mgr.tick()
			}
		}
	}()

	return mgr
}

// To be called by TTL manager's coroutine only.
func (mgr *ttlmanager) tick() {
	mgr.Lock()
	defer mgr.Unlock()

	now := time.Now()
	batch, err := mgr.ds.Batch()
	if err != nil {
		log.Error(err)
		return
	}
	for key, entry := range mgr.entries {
		if entry.ExpiresAt.Before(now) {
			if err := batch.Delete(key); err != nil {
				log.Error(err)
			} else {
				mgr.cache.Remove(key)
			}
			delete(mgr.entries, key)
		}
	}
	err = batch.Commit()
	if err != nil {
		log.Error(err)
	}
}

func (mgr *ttlmanager) setTTLs(keys []ds.Key, ttl time.Duration, add bool) {
	mgr.Lock()
	defer mgr.Unlock()

	expiration := time.Now().Add(ttl)
	for _, key := range keys {
		update := true
		if add {
			if entry, ok := mgr.entries[key]; ok {
				if entry.ExpiresAt.After(expiration) {
					update = false
				}
			}
		}
		if update {
			if ttl <= 0 {
				delete(mgr.entries, key)
			} else {
				mgr.entries[key] = &ttlentry{TTL: ttl, ExpiresAt: expiration}
			}
		}
	}
}

func (mgr *ttlmanager) updateTTLs(prefix ds.Key, oldTTL, newTTL time.Duration) {
	mgr.Lock()
	defer mgr.Unlock()

	now := time.Now()
	var keys []ds.Key
	for key, entry := range mgr.entries {
		if key.IsDescendantOf(prefix) && entry.TTL == oldTTL {
			keys = append(keys, key)
			entry.TTL = newTTL
			entry.ExpiresAt = now.Add(newTTL)
		}
	}
}

func (mgr *ttlmanager) clear(prefix ds.Key) {
	mgr.Lock()
	defer mgr.Unlock()

	for key := range mgr.entries {
		if key.IsDescendantOf(prefix) {
			delete(mgr.entries, key)
		}
	}
}
