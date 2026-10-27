package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/waku"
)

type fakeMissingDependencyFetcher struct {
	mu        sync.Mutex
	processed map[string]bool
	fetches   [][]string
	// fetchErrs[i] is returned by the i-th fetch; fetches beyond it succeed.
	fetchErrs []error
	// onFetch, if set, runs on every fetch (e.g. to mark hashes as processed).
	onFetch func(attempt int)
}

func (f *fakeMissingDependencyFetcher) AlreadyProcessed(hashes []string) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	hits := make(map[string]bool)
	for _, h := range hashes {
		if f.processed[h] {
			hits[h] = true
		}
	}
	return hits, nil
}

func (f *fakeMissingDependencyFetcher) FetchMessagesByHashes(_ context.Context, hashes []string) error {
	f.mu.Lock()
	attempt := len(f.fetches)
	f.fetches = append(f.fetches, append([]string(nil), hashes...))
	onFetch := f.onFetch
	f.mu.Unlock()
	if onFetch != nil {
		onFetch(attempt)
	}
	if attempt < len(f.fetchErrs) {
		return f.fetchErrs[attempt]
	}
	return nil
}

func (f *fakeMissingDependencyFetcher) markProcessed(h string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.processed == nil {
		f.processed = make(map[string]bool)
	}
	f.processed[h] = true
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestMissingDepsFetcher(f missingDependencyFetcher) (*missingDepsFetcher, *fakeClock) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	requested := newRequestedHashes(time.Hour, 100)
	requested.now = clock.Now
	fetcher := newMissingDepsFetcher(zap.NewNop(), f, requested)
	fetcher.now = clock.Now
	return fetcher, clock
}

// step runs one worker iteration at the current fake time and reports whether
// a batch was taken.
func step(t *testing.T, m *missingDepsFetcher) bool {
	t.Helper()
	batch, _ := m.takeBatch(m.now())
	if len(batch) == 0 {
		return false
	}
	m.fetchBatch(context.Background(), batch)
	return true
}

func TestMissingDepsFetcher_WaitsBatchDelay(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	require.False(t, step(t, m))

	clock.advance(missingDepsBatchDelay)
	require.True(t, step(t, m))
	require.Equal(t, [][]string{{"0x01"}}, f.fetches)
	require.Zero(t, m.pendingCount())
}

func TestMissingDepsFetcher_BatchesCallbacksIntoOneQuery(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01", "0x02"})
	clock.advance(time.Second)
	m.enqueue([]string{"0x02", "0x03"})
	clock.advance(time.Second)

	require.True(t, step(t, m))
	require.False(t, step(t, m))
	require.Equal(t, [][]string{{"0x01", "0x02", "0x03"}}, f.fetches)
}

func TestMissingDepsFetcher_RespectsMaxBatch(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)
	m.maxBatch = 2

	m.enqueue([]string{"0x01", "0x02", "0x03"})
	clock.advance(missingDepsBatchDelay)

	require.True(t, step(t, m))
	require.True(t, step(t, m))
	require.False(t, step(t, m))
	require.Equal(t, [][]string{{"0x01", "0x02"}, {"0x03"}}, f.fetches)
}

func TestMissingDepsFetcher_SkipsRecentlyRequested(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01", "0x02"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	// 0x01/0x02 were not processed (e.g. failed processing or not in store), but
	// must not be requested again; only the new hash is fetched.
	queued, _ := m.enqueue([]string{"0x01", "0x02", "0x03"})
	require.Equal(t, 1, queued)
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	require.Equal(t, [][]string{{"0x01", "0x02"}, {"0x03"}}, f.fetches)
}

func TestMissingDepsFetcher_SkipsAlreadyProcessed(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	f.markProcessed("0x01")
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	require.Empty(t, f.fetches)
	require.Zero(t, m.pendingCount())
}

func TestMissingDepsFetcher_RetriesWithBackoff(t *testing.T) {
	eof := errors.New("EOF")
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{eof}}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	clock.advance(missingDepsRetryDelays[0] - time.Millisecond)
	require.False(t, step(t, m))
	clock.advance(time.Millisecond)
	require.True(t, step(t, m))

	require.Equal(t, [][]string{{"0x01"}, {"0x01"}}, f.fetches)
	require.Zero(t, m.pendingCount())
}

func TestMissingDepsFetcher_TimeoutUsesLongerBackoff(t *testing.T) {
	timeout := fmt.Errorf("%w: %w", waku.ErrStorenodeTimeout, context.DeadlineExceeded)
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{timeout}}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	clock.advance(missingDepsRetryDelays[0])
	require.False(t, step(t, m))
	clock.advance(missingDepsTimeoutRetryDelays[0] - missingDepsRetryDelays[0])
	require.True(t, step(t, m))
	require.Len(t, f.fetches, 2)
}

func TestMissingDepsFetcher_RetrySkipsAlreadyReceived(t *testing.T) {
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{errors.New("EOF")}}
	// The first attempt delivers 0x01 before the connection drops.
	f.onFetch = func(attempt int) {
		if attempt == 0 {
			f.markProcessed("0x01")
		}
	}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01", "0x02"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)
	clock.advance(missingDepsRetryDelays[0])
	step(t, m)

	require.Equal(t, [][]string{{"0x01", "0x02"}, {"0x02"}}, f.fetches)
}

func TestMissingDepsFetcher_NewHashesJoinDueRetry(t *testing.T) {
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{errors.New("EOF")}}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	clock.advance(missingDepsRetryDelays[0])
	m.enqueue([]string{"0x02"})
	require.True(t, step(t, m))

	require.Equal(t, [][]string{{"0x01"}, {"0x01", "0x02"}}, f.fetches)
}

func TestMissingDepsFetcher_GivesUpThenAllowsLaterRequest(t *testing.T) {
	eof := errors.New("EOF")
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{eof, eof, eof, eof}}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)
	for _, d := range missingDepsRetryDelays {
		clock.advance(d)
		require.True(t, step(t, m))
	}
	require.Len(t, f.fetches, len(missingDepsRetryDelays)+1)
	require.Zero(t, m.pendingCount())

	// Exhausted hashes are not re-requested right away...
	queued, _ := m.enqueue([]string{"0x01"})
	require.Zero(t, queued)
	// ...but sooner than successfully requested ones.
	clock.advance(missingDepsExhaustedTTL)
	queued, _ = m.enqueue([]string{"0x01"})
	require.Equal(t, 1, queued)
}

func TestMissingDepsFetcher_DropsOverCapacity(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)
	m.maxQueue = 2

	queued, dropped := m.enqueue([]string{"0x01", "0x02", "0x03"})
	require.Equal(t, 2, queued)
	require.Equal(t, 1, dropped)

	clock.advance(missingDepsBatchDelay)
	step(t, m)

	// The dropped hash was never requested, so it can be queued again.
	queued, _ = m.enqueue([]string{"0x03"})
	require.Equal(t, 1, queued)
}

func TestMissingDepsFetcher_CancelledFetchKeepsPending(t *testing.T) {
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{context.Canceled}}
	m, clock := newTestMissingDepsFetcher(f)
	ctx, cancel := context.WithCancel(context.Background())
	f.onFetch = func(int) { cancel() }

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	batch, _ := m.takeBatch(m.now())
	m.fetchBatch(ctx, batch)

	require.Equal(t, 1, m.pendingCount())
}

func TestMissingDepsFetcher_RunSingleWorkerFetchesEachHashOnce(t *testing.T) {
	var inFlight, maxInFlight int
	var inFlightMu sync.Mutex
	f := &fakeMissingDependencyFetcher{}
	f.onFetch = func(int) {
		inFlightMu.Lock()
		inFlight++
		maxInFlight = max(maxInFlight, inFlight)
		inFlightMu.Unlock()
		time.Sleep(5 * time.Millisecond)
		inFlightMu.Lock()
		inFlight--
		inFlightMu.Unlock()
	}
	m := newMissingDepsFetcher(zap.NewNop(), f, newRequestedHashes(time.Hour, 100))
	m.batchDelay = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer panics.LogOnPanic()
		defer close(done)
		m.run(ctx)
	}()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer panics.LogOnPanic()
			defer wg.Done()
			m.enqueue([]string{"0x01", "0x02", "0x03"})
		}()
	}
	wg.Wait()

	require.Eventually(t, func() bool { return m.pendingCount() == 0 }, 5*time.Second, 5*time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run did not stop after context cancellation")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	counts := make(map[string]int)
	for _, fetch := range f.fetches {
		for _, h := range fetch {
			counts[h]++
		}
	}
	require.Equal(t, map[string]int{"0x01": 1, "0x02": 1, "0x03": 1}, counts)
	require.Len(t, f.fetches, 1)
	require.Equal(t, 1, maxInFlight)
}

func TestRequestedHashes_ExpiresAfterTTL(t *testing.T) {
	now := time.Unix(0, 0)
	r := newRequestedHashes(time.Minute, 100)
	r.now = func() time.Time { return now }

	require.Equal(t, []string{"0x01"}, r.claim([]string{"0x01"}))
	require.Empty(t, r.claim([]string{"0x01"}))

	now = now.Add(time.Minute)
	require.Equal(t, []string{"0x01"}, r.claim([]string{"0x01"}))
	require.Equal(t, 1, r.len())
}

func TestRequestedHashes_EvictsOldestWhenFull(t *testing.T) {
	now := time.Unix(0, 0)
	r := newRequestedHashes(time.Hour, 2)
	r.now = func() time.Time { return now }

	r.claim([]string{"0x01"})
	now = now.Add(time.Second)
	r.claim([]string{"0x02"})
	now = now.Add(time.Second)
	r.claim([]string{"0x03"})

	require.Equal(t, 2, r.len())
	require.Equal(t, []string{"0x01"}, r.claim([]string{"0x01"}))
	require.Empty(t, r.claim([]string{"0x03"}))
}

func TestRequestedHashes_NilClaimsEverything(t *testing.T) {
	var r *requestedHashes
	require.Equal(t, []string{"0x01", "0x02"}, r.claim([]string{"0x01", "0x02"}))
}

func TestRequestedHashes_ReleaseAndExpireIn(t *testing.T) {
	now := time.Unix(0, 0)
	r := newRequestedHashes(time.Hour, 100)
	r.now = func() time.Time { return now }

	r.claim([]string{"0x01", "0x02"})
	r.release([]string{"0x01"})
	require.Equal(t, []string{"0x01"}, r.claim([]string{"0x01"}))

	r.expireIn([]string{"0x02"}, time.Minute)
	require.Empty(t, r.claim([]string{"0x02"}))
	now = now.Add(time.Minute)
	require.Equal(t, []string{"0x02"}, r.claim([]string{"0x02"}))
}
