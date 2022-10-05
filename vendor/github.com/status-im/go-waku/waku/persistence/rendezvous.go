package persistence

import (
	rendezvous "github.com/status-im/go-waku-rendezvous"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// RendezVousLevelDB is a struct used to hold a reference to a LevelDB database
type RendezVousLevelDB struct {
	db *leveldb.DB
}

// NewRendezVousLevelDB opens a LevelDB database to be used for rendezvous protocol
func NewRendezVousLevelDB(dBPath string) (*RendezVousLevelDB, error) {
	db, err := leveldb.OpenFile(dBPath, &opt.Options{OpenFilesCacheCapacity: 3})

	if err != nil {
		return nil, err
	}

	return &RendezVousLevelDB{db}, nil
}

// Delete removes a key from the database
func (r *RendezVousLevelDB) Delete(key []byte) error {
	return r.db.Delete(key, nil)
}

// Put inserts or updates a key in the database
func (r *RendezVousLevelDB) Put(key []byte, value []byte) error {
	return r.db.Put(key, value, nil)
}

// NewIterator returns an interator that can be used to iterate over all
// the records contained in the DB
func (r *RendezVousLevelDB) NewIterator(prefix []byte) rendezvous.Iterator {
	return r.db.NewIterator(util.BytesPrefix(prefix), nil)
}
