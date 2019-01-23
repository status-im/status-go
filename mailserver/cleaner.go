package mailserver

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	dbCleanerBatchSize = 1000
	dbCleanerPeriod    = time.Hour
)

// dbCleaner removes old messages from a db.
type dbCleaner struct {
	sync.RWMutex

	db        dbImpl
	batchSize int
	retention time.Duration

	period time.Duration
	cancel chan struct{}
}

// newDBCleaner returns a new cleaner for db.
func newDBCleaner(db dbImpl, retention time.Duration) *dbCleaner {
	return &dbCleaner{
		db:        db,
		retention: retention,

		batchSize: dbCleanerBatchSize,
		period:    dbCleanerPeriod,
	}
}

// Start starts a loop that cleans up old messages.
func (c *dbCleaner) Start() {
	log.Info("Starting cleaning envelopes", "period", c.period, "retention", c.retention)

	cancel := make(chan struct{})

	c.Lock()
	c.cancel = cancel
	c.Unlock()

	go c.schedule(c.period, cancel)
}

// Stops stops the cleaning loop.
func (c *dbCleaner) Stop() {
	c.Lock()
	defer c.Unlock()

	if c.cancel == nil {
		return
	}
	close(c.cancel)
	c.cancel = nil
}

func (c *dbCleaner) schedule(period time.Duration, cancel <-chan struct{}) {
	t := time.NewTicker(period)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			count, err := c.PruneEntriesOlderThan(time.Now().Add(-c.retention))
			if err != nil {
				log.Error("failed to prune data", "err", err)
			}
			log.Info("Prunned some some messages successfully", "count", count)
		case <-cancel:
			return
		}
	}
}

// PruneEntriesOlderThan removes messages sent between lower and upper timestamps
// and returns how many have been removed.
func (c *dbCleaner) PruneEntriesOlderThan(t time.Time) (int, error) {
	var zero common.Hash
	kl := NewDBKey(0, zero)
	ku := NewDBKey(uint32(t.Unix()), zero)
	i := c.db.NewIterator(&util.Range{Start: kl.Bytes(), Limit: ku.Bytes()}, nil)
	defer i.Release()

	return c.prune(i)
}

func (c *dbCleaner) prune(i iterator.Iterator) (int, error) {
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
