package log

import (
	"errors"
	"sync"
	"time"
)

// errors.
var (
	ErrBatchEmitterClosed = errors.New("batcher already closed")
)

// EmitterFunction defines a function type which is used to process a batch of Entry.
type EmitterFunction func([]Entry) error

// BatchEmitter defines a structure which collects Entry in batch
// mode until a provide size threshold is met then it's provided against
// a provided function for procesing.
type BatchEmitter struct {
	maxlen     int
	maxwait    time.Duration
	closed     bool
	entrybatch []Entry
	fnError    chan error
	entries    chan Entry
	stop       chan struct{}
	fn         EmitterFunction
	wg         sync.WaitGroup
}

// BatchEmit returns a new instance of a BatchEmitter.
func BatchEmit(maxSize int, maxwait time.Duration, fn EmitterFunction) *BatchEmitter {
	var batch BatchEmitter
	batch.fn = fn
	batch.maxwait = maxwait
	batch.maxlen = maxSize
	batch.stop = make(chan struct{}, 0)
	batch.entries = make(chan Entry, 0)
	batch.fnError = make(chan error, 1)

	batch.wg.Add(1)
	go batch.manage()

	return &batch
}

// Emit takes provided entries and emits giving entries into batch loop.
func (bm *BatchEmitter) Emit(en Entry) error {
	select {
	case <-bm.stop:
		return ErrBatchEmitterClosed
	case bm.entries <- en:
		select {
		case err := <-bm.fnError:
			return err
		case <-time.After(3 * time.Millisecond):
			return nil
		}
	}
}

// Close ends the operation of the batch emitter and releases all associated resources.
func (bm *BatchEmitter) Close() error {
	if bm.closed {
		return ErrBatchEmitterClosed
	}

	close(bm.stop)
	bm.wg.Wait()

	bm.closed = true
	return nil
}

func (bm *BatchEmitter) manage() {
	defer bm.wg.Done()

	ticker := time.NewTimer(bm.maxwait)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ticker.Reset(bm.maxwait)

			if len(bm.entrybatch) == 0 {
				continue
			}

			batch := bm.entrybatch
			bm.entrybatch = nil

			if err := bm.fn(batch); err != nil {
				bm.fnError <- err
			}

		case entry, ok := <-bm.entries:
			if !ok {
				return
			}

			ticker.Reset(bm.maxwait)

			bm.entrybatch = append(bm.entrybatch, entry)

			if len(bm.entrybatch) >= bm.maxlen {
				batch := bm.entrybatch
				bm.entrybatch = nil

				if err := bm.fn(batch); err != nil {
					bm.fnError <- err
				}
				continue
			}

		case <-bm.stop:
			return
		}
	}
}
