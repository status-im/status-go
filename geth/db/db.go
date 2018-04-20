package db

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/log"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

type storagePrefix byte

const (
	// PeersCache is used for the db entries used for peers DB
	PeersCache storagePrefix = iota
	// DeduplicatorCache is used for the db entries used for messages
	// deduplication cache
	DeduplicatorCache
)

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
	opts := &opt.Options{OpenFilesCacheCapacity: 5}
	db, err := leveldb.OpenFile(path, opts)
	if _, iscorrupted := err.(*errors.ErrCorrupted); iscorrupted {
		log.Info("database is corrupted trying to recover", "path", path)
		db, err = leveldb.RecoverFile(path, nil)
	}
	if err != nil {
		return nil, err
	}
	return db, err
}
