package pstoreds

import (
	"github.com/pkg/errors"

	ds "github.com/ipfs/go-datastore"
)

// how many operations are queued in a cyclic batch before we flush it.
var defaultOpsPerCyclicBatch = 20

// cyclicBatch buffers ds write operations and automatically flushes them after defaultOpsPerCyclicBatch (20) have been
// queued. An explicit `Commit()` closes this cyclic batch, erroring all further operations.
//
// It is similar to go-ds autobatch, but it's driven by an actual Batch facility offered by the
// ds.
type cyclicBatch struct {
	threshold int
	ds.Batch
	ds      ds.Batching
	pending int
}

func newCyclicBatch(ds ds.Batching, threshold int) (ds.Batch, error) {
	batch, err := ds.Batch()
	if err != nil {
		return nil, err
	}
	return &cyclicBatch{Batch: batch, ds: ds}, nil
}

func (cb *cyclicBatch) cycle() (err error) {
	if cb.Batch == nil {
		return errors.New("cyclic batch is closed")
	}
	if cb.pending < cb.threshold {
		// we haven't reached the threshold yet.
		return nil
	}
	// commit and renew the batch.
	if err = cb.Batch.Commit(); err != nil {
		return errors.Wrap(err, "failed while committing cyclic batch")
	}
	if cb.Batch, err = cb.ds.Batch(); err != nil {
		return errors.Wrap(err, "failed while renewing cyclic batch")
	}
	return nil
}

func (cb *cyclicBatch) Put(key ds.Key, val []byte) error {
	if err := cb.cycle(); err != nil {
		return err
	}
	cb.pending++
	return cb.Batch.Put(key, val)
}

func (cb *cyclicBatch) Delete(key ds.Key) error {
	if err := cb.cycle(); err != nil {
		return err
	}
	cb.pending++
	return cb.Batch.Delete(key)
}

func (cb *cyclicBatch) Commit() error {
	if cb.Batch == nil {
		return errors.New("cyclic batch is closed")
	}
	if err := cb.Batch.Commit(); err != nil {
		return err
	}
	cb.pending = 0
	cb.Batch = nil
	return nil
}
