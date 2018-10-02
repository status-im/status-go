package ratelimiter

import (
	"encoding/binary"
	time "time"

	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	BlacklistBucket byte = 1 + iota
	CapacityBucket
)

// DBInterface defines leveldb methods used by ratelimiter.
type DBInterface interface {
	Put(key, value []byte, wo *opt.WriteOptions) error
	Get(key []byte, ro *opt.ReadOptions) (value []byte, err error)
	Delete(key []byte, wo *opt.WriteOptions) error
}

func WithPrefix(db DBInterface, prefix []byte) IsolatedDB {
	return IsolatedDB{db: db, prefix: prefix}
}

type IsolatedDB struct {
	db     DBInterface
	prefix []byte
}

func (db IsolatedDB) withPrefix(key []byte) []byte {
	fkey := make([]byte, len(db.prefix)+len(key))
	copy(fkey, db.prefix)
	copy(fkey[len(db.prefix):], key)
	return fkey
}

func (db IsolatedDB) Put(key, value []byte, wo *opt.WriteOptions) error {
	return db.db.Put(db.withPrefix(key), value, wo)
}

func (db IsolatedDB) Get(key []byte, ro *opt.ReadOptions) (value []byte, err error) {
	return db.db.Get(db.withPrefix(key), ro)
}

func (db IsolatedDB) Delete(key []byte, wo *opt.WriteOptions) error {
	return db.db.Delete(db.withPrefix(key), wo)
}

// BlacklistRecord is a record with information of a deadline for a particular ID.
type BlacklistRecord struct {
	ID       []byte
	Deadline time.Time
}

func (r BlacklistRecord) Key() []byte {
	key := make([]byte, len(r.ID)+1)
	key[0] = BlacklistBucket
	copy(key[1:], r.ID)
	return key
}

func (r BlacklistRecord) Write(db DBInterface) error {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], uint64(r.Deadline.Unix()))
	return db.Put(r.Key(), buf[:], nil)
}

func (r *BlacklistRecord) Read(db DBInterface) error {
	val, err := db.Get(r.Key(), nil)
	if err != nil {
		return err
	}
	deadline := binary.BigEndian.Uint64(val)
	r.Deadline = time.Unix(int64(deadline), 0)
	return nil
}

func (r BlacklistRecord) Remove(db DBInterface) error {
	return db.Delete(r.Key(), nil)
}

// CapacityRecord tracks how much was taken from a bucket and a time.
type CapacityRecord struct {
	ID        []byte
	Taken     int64
	Timestamp time.Time
}

func (r CapacityRecord) Key() []byte {
	key := make([]byte, len(r.ID)+1)
	key[0] = CapacityBucket
	copy(key[1:], r.ID)
	return key
}

func (r CapacityRecord) Write(db DBInterface) error {
	buf := [16]byte{}
	binary.BigEndian.PutUint64(buf[:], uint64(r.Taken))
	binary.BigEndian.PutUint64(buf[8:], uint64(r.Timestamp.Unix()))
	return db.Put(r.Key(), buf[:], nil)
}

func (r *CapacityRecord) Read(db DBInterface) error {
	val, err := db.Get(r.Key(), nil)
	if err != nil {
		return err
	}
	r.Taken = int64(binary.BigEndian.Uint64(val[:8]))
	r.Timestamp = time.Unix(int64(binary.BigEndian.Uint64(val[8:])), 0)
	return nil
}
