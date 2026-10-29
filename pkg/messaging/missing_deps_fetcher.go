package messaging

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/messaging/waku"
)

const (
	// missingDepsBatchDelay is how long new hashes wait for others to join
	// their batch, and the minimum pause between two by-hash store queries.
	missingDepsBatchDelay = 2 * time.Second
	missingDepsMaxBatch   = 50
	// missingDepsMaxQueue caps hashes waiting to be fetched; extra ones are dropped.
	missingDepsMaxQueue = 1000
	// missingDepsFetchTimeout bounds one batch fetch, across storenode failovers.
	missingDepsFetchTimeout = 70 * time.Second
	// missingDepsExhaustedTTL is how long a hash that failed all its attempts
	// stays unrequestable, instead of requestedHashesTTL.
	missingDepsExhaustedTTL = 5 * time.Minute
)

// missingDepsRetryDelays are the waits before each retry of a hash whose fetch
// failed. SDS only re-detects a missing dependency when a later message
// references it again, so a fetch that is not retried is usually lost for good.
var missingDepsRetryDelays = []time.Duration{5 * time.Second, 15 * time.Second, 45 * time.Second}

// missingDepsTimeoutRetryDelays replace missingDepsRetryDelays after a
// storenode timeout: the storenode usually keeps processing the timed-out
// query, so retrying soon only adds load.
var missingDepsTimeoutRetryDelays = []time.Duration{30 * time.Second, 2 * time.Minute, 2 * time.Minute}

type missingDependencyFetcher interface {
	AlreadyProcessed(messageHashes []string) (map[string]bool, error)
	// EnvelopesSeen reports hashes recently handed on for processing, even if
	// processing failed. Fetching them again would only redeliver the same
	// envelope.
	EnvelopesSeen(messageHashes []string) map[string]bool
	FetchMessagesByHashes(ctx context.Context, messageHashes []string) error
}

type pendingMissingDep struct {
	// attempts is the number of failed fetches so far.
	attempts int
	dueAt    time.Time
	seq      uint64
}

// missingDepsFetcher fetches SDS missing dependencies from storenodes with a
// single worker: hashes are deduplicated (in flight and recently requested),
// batched into one by-hash query and retried with per-hash backoff.
type missingDepsFetcher struct {
	logger    *zap.Logger
	fetcher   missingDependencyFetcher
	requested *requestedHashes
	now       func() time.Time

	batchDelay         time.Duration
	maxBatch           int
	maxQueue           int
	fetchTimeout       time.Duration
	retryDelays        []time.Duration
	timeoutRetryDelays []time.Duration
	exhaustedTTL       time.Duration

	mu      sync.Mutex
	pending map[string]*pendingMissingDep
	seq     uint64
	wake    chan struct{}
}

func newMissingDepsFetcher(logger *zap.Logger, fetcher missingDependencyFetcher, requested *requestedHashes) *missingDepsFetcher {
	return &missingDepsFetcher{
		logger:             logger,
		fetcher:            fetcher,
		requested:          requested,
		now:                time.Now,
		batchDelay:         missingDepsBatchDelay,
		maxBatch:           missingDepsMaxBatch,
		maxQueue:           missingDepsMaxQueue,
		fetchTimeout:       missingDepsFetchTimeout,
		retryDelays:        missingDepsRetryDelays,
		timeoutRetryDelays: missingDepsTimeoutRetryDelays,
		exhaustedTTL:       missingDepsExhaustedTTL,
		pending:            make(map[string]*pendingMissingDep),
		wake:               make(chan struct{}, 1),
	}
}

// enqueue schedules hashes for fetching. It does not block: hashes already
// received, requested recently or already queued are skipped, and hashes
// beyond the queue capacity are dropped.
func (f *missingDepsFetcher) enqueue(hashes []string) (queued, dropped int) {
	claimed := f.requested.claim(withoutHashes(hashes, f.fetcher.EnvelopesSeen(hashes)))
	if len(claimed) == 0 {
		return 0, 0
	}

	var droppedHashes []string
	f.mu.Lock()
	dueAt := f.now().Add(f.batchDelay)
	for _, h := range claimed {
		if _, ok := f.pending[h]; ok {
			continue
		}
		if len(f.pending) >= f.maxQueue {
			droppedHashes = append(droppedHashes, h)
			continue
		}
		f.seq++
		f.pending[h] = &pendingMissingDep{dueAt: dueAt, seq: f.seq}
		queued++
	}
	f.mu.Unlock()

	// Dropped hashes were never requested, so a later reference may retry them.
	f.requested.release(droppedHashes)

	if queued > 0 {
		select {
		case f.wake <- struct{}{}:
		default:
		}
	}
	return queued, len(droppedHashes)
}

// takeBatch returns up to maxBatch hashes to fetch now, oldest first. Once
// any hash is due, hashes still waiting for their first attempt join the
// batch early. It also returns when the next remaining hash is due, or zero.
func (f *missingDepsFetcher) takeBatch(now time.Time) (batch []string, nextDue time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()

	anyDue := false
	for _, p := range f.pending {
		if !p.dueAt.After(now) {
			anyDue = true
			break
		}
	}

	if anyDue {
		for h, p := range f.pending {
			if !p.dueAt.After(now) || p.attempts == 0 {
				batch = append(batch, h)
			}
		}
		sort.Slice(batch, func(i, j int) bool { return f.pending[batch[i]].seq < f.pending[batch[j]].seq })
		if len(batch) > f.maxBatch {
			batch = batch[:f.maxBatch]
		}
	}

	inBatch := make(map[string]bool, len(batch))
	for _, h := range batch {
		inBatch[h] = true
	}
	for h, p := range f.pending {
		if inBatch[h] {
			continue
		}
		if nextDue.IsZero() || p.dueAt.Before(nextDue) {
			nextDue = p.dueAt
		}
	}
	return batch, nextDue
}

// fetchBatch makes a single by-hash store query for the hashes of batch that
// were not received in the meantime, and reschedules them if it fails.
func (f *missingDepsFetcher) fetchBatch(ctx context.Context, batch []string) {
	alreadyProcessed, err := f.fetcher.AlreadyProcessed(batch)
	if err != nil {
		f.logger.Debug("failed to check missing dependencies cache", zap.Error(err))
	}
	received := make(map[string]bool, len(alreadyProcessed))
	for h, ok := range alreadyProcessed {
		received[h] = ok
	}
	for h, ok := range f.fetcher.EnvelopesSeen(batch) {
		received[h] = received[h] || ok
	}

	toFetch := make([]string, 0, len(batch))
	f.mu.Lock()
	for _, h := range batch {
		if received[h] {
			delete(f.pending, h)
		} else {
			toFetch = append(toFetch, h)
		}
	}
	f.mu.Unlock()
	if len(toFetch) == 0 {
		return
	}

	fetchStart := time.Now()
	fetchCtx, cancel := context.WithTimeout(ctx, f.fetchTimeout)
	err = f.fetcher.FetchMessagesByHashes(fetchCtx, toFetch)
	cancel()
	f.logger.Debug("fetched missing dependencies from storenode",
		zap.Int("batch", len(batch)),
		zap.Int("skippedReceived", len(batch)-len(toFetch)),
		zap.Strings("hashes", toFetch),
		zap.Duration("duration", time.Since(fetchStart)),
		zap.Error(err),
	)

	if ctx.Err() != nil {
		return
	}

	delays := f.retryDelays
	if errors.Is(err, waku.ErrStorenodeTimeout) || errors.Is(err, context.DeadlineExceeded) {
		delays = f.timeoutRetryDelays
	}

	var exhausted []string
	now := f.now()
	f.mu.Lock()
	for _, h := range toFetch {
		p, ok := f.pending[h]
		if !ok {
			continue
		}
		if err == nil {
			delete(f.pending, h)
			continue
		}
		p.attempts++
		if p.attempts > len(delays) {
			delete(f.pending, h)
			exhausted = append(exhausted, h)
			continue
		}
		p.dueAt = now.Add(delays[p.attempts-1])
	}
	f.mu.Unlock()

	if len(exhausted) > 0 {
		f.requested.expireIn(exhausted, f.exhaustedTTL)
		f.logger.Warn("failed to fetch missing dependencies from storenode",
			zap.Strings("missingDeps", exhausted),
			zap.Error(err),
		)
	}
}

// run is the single worker loop. It returns when ctx is done.
func (f *missingDepsFetcher) run(ctx context.Context) {
	for {
		batch, nextDue := f.takeBatch(f.now())
		if len(batch) > 0 {
			f.fetchBatch(ctx, batch)
			// Space out store queries, and let new hashes join the next batch.
			if !sleepCtx(ctx, f.batchDelay) {
				return
			}
			continue
		}

		var timer *time.Timer
		var timerC <-chan time.Time
		if !nextDue.IsZero() {
			timer = time.NewTimer(nextDue.Sub(f.now()))
			timerC = timer.C
		}
		select {
		case <-ctx.Done():
		case <-f.wake:
		case <-timerC:
		}
		if timer != nil {
			timer.Stop()
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (f *missingDepsFetcher) pendingCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pending)
}

// withoutHashes returns the hashes not in exclude, keeping their order.
func withoutHashes(hashes []string, exclude map[string]bool) []string {
	if len(exclude) == 0 {
		return hashes
	}
	kept := make([]string, 0, len(hashes))
	for _, h := range hashes {
		if !exclude[h] {
			kept = append(kept, h)
		}
	}
	return kept
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
