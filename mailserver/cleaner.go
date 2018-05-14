package mailserver

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const batchSize = 1000

// Cleaner removes old messages from a db
type Cleaner struct {
	db        *leveldb.DB
	batchSize int
}

// NewCleanerWithDB returns a new Cleaner for db
func NewCleanerWithDB(db *leveldb.DB) *Cleaner {
	return &Cleaner{
		db:        db,
		batchSize: batchSize,
	}
}

// Prune removes messages sent between lower and upper timestamps and returns how many has been removed
func (c *Cleaner) Prune(lower, upper uint32) (int, error) {
	var zero common.Hash
	kl := NewDbKey(lower, zero)
	ku := NewDbKey(upper, zero)
	i := c.db.NewIterator(&util.Range{Start: kl.raw, Limit: ku.raw}, nil)
	defer i.Release()

	return c.prune(i)
}

func (c *Cleaner) prune(i iterator.Iterator) (int, error) {
	batch := leveldb.Batch{}
	removed := 0

	for i.Next() {
		batch.Delete(i.Key())

		if batch.Len() == c.batchSize {
			if err := c.db.Write(&batch, nil); err != nil {
				return removed, err
			}

			removed = removed + batch.Len()
			batch.Reset()
		}
	}

	if batch.Len() > 0 {
		if err := c.db.Write(&batch, nil); err != nil {
			return removed, err
		}

		removed = removed + batch.Len()
	}

	return removed, nil
}
