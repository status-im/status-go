package mailserver

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	cleanerBatchSize = 1000
	cleanerPeriod    = time.Hour
)

// cleaner removes old messages from a db.
type cleaner struct {
	sync.RWMutex

	db        dbImpl
	batchSize int
	retention time.Duration

	period time.Duration
	cancel chan struct{}
}

// NewCleanerWithDB returns a new cleaner for db.
func newCleanerWithDB(db dbImpl, retention time.Duration) *cleaner {
	return &cleaner{
		db:        db,
		retention: retention,

		batchSize: cleanerBatchSize,
		period:    cleanerPeriod,
	}
}

// Start starts a loop that cleans up old messages.
func (c *cleaner) Start() {
	cancel := make(chan struct{})

	c.Lock()
	c.cancel = cancel
	c.Unlock()

	go c.schedule(c.period, cancel)
}

// Stops stops the cleaning loop.
func (c *cleaner) Stop() {
	c.Lock()
	defer c.Unlock()

	if c.cancel == nil {
		return
	}
	close(c.cancel)
	c.cancel = nil
}

func (c *cleaner) schedule(period time.Duration, cancel <-chan struct{}) {
	t := time.NewTicker(period)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			lower := uint32(0)
			upper := uint32(time.Now().Add(-c.retention).Unix())
			if _, err := c.Prune(lower, upper); err != nil {
				log.Error("failed to prune data", "err", err)
			}
		case <-cancel:
			return
		}
	}
}

// Prune removes messages sent between lower and upper timestamps
// and returns how many have been removed.
func (c *cleaner) Prune(lower, upper uint32) (int, error) {
	fmt.Printf("============= prune from %d to %d\n", lower, upper)

	var zero common.Hash
	kl := NewDBKey(lower, zero)
	ku := NewDBKey(upper, zero)
	i := c.db.NewIterator(&util.Range{Start: kl.Bytes(), Limit: ku.Bytes()}, nil)
	defer i.Release()

	return c.prune(i)
}

func (c *cleaner) prune(i iterator.Iterator) (int, error) {
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
