package db

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/log"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type storagePrefix byte

const (
	// PeersCache is used for the db entries used for peers DB
	PeersCache storagePrefix = iota
	// DeduplicatorCache is used for the db entries used for messages
	// deduplication cache
	DeduplicatorCache
	// MailserversCache is a list of mail servers provided by users.
	MailserversCache
	// TopicHistoryBucket isolated bucket for storing history metadata.
	TopicHistoryBucket
	// HistoryRequestBucket isolated bucket for storing list of pending requests.
	HistoryRequestBucket
)

// NewMemoryDB returns leveldb with memory backend prefixed with a bucket.
func NewMemoryDB() (*leveldb.DB, error) {
	return leveldb.Open(storage.NewMemStorage(), nil)
}

// NewIsolatedDB returns instance that ensures isolated operations.
func NewIsolatedDB(db *leveldb.DB, prefix storagePrefix) PrefixedLevelDB {
	return PrefixedLevelDB{
		db:     db,
		prefix: prefix,
	}
}

// NewIsolatedMemoryDB wraps in memory leveldb with provided bucket.
func NewIsolatedMemoryDB(prefix storagePrefix) (pdb PrefixedLevelDB, err error) {
	db, err := NewMemoryDB()
	if err != nil {
		return pdb, err
	}
	return NewIsolatedDB(db, prefix), nil
}

// Key creates a DB key for a specified service with specified data
func Key(prefix storagePrefix, data ...[]byte) []byte {
	keyLength := 1
	for _, d := range data {
		keyLength += len(d)
	}
	key := make([]byte, keyLength)
	key[0] = byte(prefix)
	startPos := 1
	for _, d := range data {
		copy(key[startPos:], d[:])
		startPos += len(d)
	}

	return key
}

// Create returns status pointer to leveldb.DB.
func Create(path, dbName string) (*leveldb.DB, error) {
	// Create euphemeral storage if the node config path isn't provided
	if path == "" {
		return leveldb.Open(storage.NewMemStorage(), nil)
	}

	path = filepath.Join(path, dbName)
	return Open(path, &opt.Options{OpenFilesCacheCapacity: 5})
}

// Open opens an existing leveldb database
func Open(path string, opts *opt.Options) (db *leveldb.DB, err error) {
	db, err = leveldb.OpenFile(path, opts)
	if _, iscorrupted := err.(*errors.ErrCorrupted); iscorrupted {
		log.Info("database is corrupted trying to recover", "path", path)
		db, err = leveldb.RecoverFile(path, nil)
	}
	return
}

// PrefixedLevelDB database where all operations will be prefixed with a certain bucket.
type PrefixedLevelDB struct {
	db     *leveldb.DB
	prefix storagePrefix
}

func (db PrefixedLevelDB) prefixedKey(key []byte) []byte {
	endkey := make([]byte, len(key)+1)
	endkey[0] = byte(db.prefix)
	copy(endkey[1:], key)
	return endkey
}

func (db PrefixedLevelDB) Put(key, value []byte) error {
	return db.db.Put(db.prefixedKey(key), value, nil)
}

func (db PrefixedLevelDB) Get(key []byte) ([]byte, error) {
	return db.db.Get(db.prefixedKey(key), nil)
}

// Range returns leveldb util.Range prefixed with a single byte.
// If prefix is nil range will iterate over all records in a given bucket.
func (db PrefixedLevelDB) Range(prefix, limit []byte) *util.Range {
	if limit == nil {
		return util.BytesPrefix(db.prefixedKey(prefix))
	}
	return &util.Range{Start: db.prefixedKey(prefix), Limit: db.prefixedKey(limit)}
}

// Delete removes key from database.
func (db PrefixedLevelDB) Delete(key []byte) error {
	return db.db.Delete(db.prefixedKey(key), nil)
}

// NewIterator returns iterator for a given slice.
func (db PrefixedLevelDB) NewIterator(slice *util.Range) PrefixedIterator {
	return PrefixedIterator{db.db.NewIterator(slice, nil)}
}

// PrefixedIterator wraps leveldb iterator, works mostly the same way.
// The only difference is that first byte of the key is dropped.
type PrefixedIterator struct {
	iter iterator.Iterator
}

// Key returns key of the current item.
func (iter PrefixedIterator) Key() []byte {
	return iter.iter.Key()[1:]
}

// Value returns actual value of the current item.
func (iter PrefixedIterator) Value() []byte {
	return iter.iter.Value()
}

// Error returns accumulated error.
func (iter PrefixedIterator) Error() error {
	return iter.iter.Error()
}

// Prev moves cursor backward.
func (iter PrefixedIterator) Prev() bool {
	return iter.iter.Prev()
}

// Next moves cursor forward.
func (iter PrefixedIterator) Next() bool {
	return iter.iter.Next()
}
