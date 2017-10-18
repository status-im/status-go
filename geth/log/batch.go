package log

import (
	"errors"
	"time"
)

// errors.
var (
	ErrBatchEmitterClosed = errors.New("batcher already closed")
)

// CommitFunction defines a function type which is used to process a batch of Entry.
type CommitFunction func([]Entry) error

// BatchEmitter defines a structure which collects Entry in batch
// mode until a provide size threshold is met then it's provided against
// a provided function for procesing.
type BatchEmitter struct {
	maxlen    int
	commitErr error
	batch     []Entry
	actions   chan func()
	maxwait   time.Duration
	fn        CommitFunction
}

// BatchEmit returns a new instance of a BatchEmitter.
func BatchEmit(maxSize int, maxwait time.Duration, fn CommitFunction) *BatchEmitter {
	var batch BatchEmitter
	batch.fn = fn
	batch.maxlen = maxSize
	batch.maxwait = maxwait
	batch.actions = make(chan func(), 0)

	return &batch
}

// Emit takes provided entries and emits giving entries into batch,
// returning any error encountered with the addition of the entry
// or one received during the last commit of the entries.
func (bm *BatchEmitter) Emit(en Entry) error {
	if bm.commitErr != nil {
		return bm.commitErr
	}

	errChan := make(chan error, 0)
	action := func() {
		bm.batch = append(bm.batch, en)

		if len(bm.batch) >= bm.maxlen {
			batch := bm.batch
			bm.batch = nil

			if err := bm.fn(batch); err != nil {
				errChan <- err
				return
			}
		}

		close(errChan)
	}

	select {
	case bm.actions <- action:
		return <-errChan
	case <-time.After(5 * time.Millisecond):
		return nil
	}
}

// Emit takes provided entries and emits giving entries into batch loop.
func (bm *BatchEmitter) commit() {
	bm.actions <- func() {
		if len(bm.batch) == 0 {
			return
		}

		batch := bm.batch
		bm.batch = nil
		bm.commitErr = bm.fn(batch)
	}
}

// Run handles running the necessary logic to add entries into the batch
// and call the necessary commit call to evaluate the provided function with
// the collected Entries.
func (bm *BatchEmitter) Run(closeChan <-chan struct{}) {
	ticker := time.NewTimer(bm.maxwait)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ticker.Reset(bm.maxwait)
			go bm.commit()
		case action := <-bm.actions:
			ticker.Reset(bm.maxwait)
			action()
		case <-closeChan:
			return
		}
	}
}
