package persistence

import (
	rendezvous "github.com/status-im/go-waku-rendezvous"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type RendezVousLevelDB struct {
	db *leveldb.DB
}

func NewRendezVousLevelDB(dBPath string) (*RendezVousLevelDB, error) {
	db, err := leveldb.OpenFile(dBPath, &opt.Options{OpenFilesCacheCapacity: 3})

	if err != nil {
		return nil, err
	}

	return &RendezVousLevelDB{db}, nil
}

func (r *RendezVousLevelDB) Delete(key []byte) error {
	return r.db.Delete(key, nil)
}

func (r *RendezVousLevelDB) Put(key []byte, value []byte) error {
	return r.db.Put(key, value, nil)
}

func (r *RendezVousLevelDB) NewIterator(prefix []byte) rendezvous.Iterator {
	return r.db.NewIterator(util.BytesPrefix(prefix), nil)
}
