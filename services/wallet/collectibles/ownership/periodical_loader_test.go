package ownership

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/pubsub"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"go.uber.org/zap/zaptest"
)

// fakeFetcher blocks until its context is cancelled, allowing tests to observe
// whether Stop properly propagates to in-flight loads.
type fakeFetcher struct {
	mu       sync.Mutex
	calls    int
	entered  chan struct{} // sent on each time a fetch begins
	returned chan struct{} // closed after each FetchCollectibleOwnershipByOwner returns
}

func newFakeFetcher() *fakeFetcher {
	return &fakeFetcher{
		entered:  make(chan struct{}, 10),
		returned: make(chan struct{}),
	}
}

func (f *fakeFetcher) FetchCollectibleOwnershipByOwner(ctx context.Context, _ walletCommon.ChainID, _ common.Address, _ string, _ int, _ string) (*thirdparty.CollectibleOwnershipContainer, error) {
	f.mu.Lock()
	f.calls++
	// Refresh the channel so each call gets its own signal.
	ch := make(chan struct{})
	f.returned = ch
	f.mu.Unlock()

	f.entered <- struct{}{}

	// Block until context is cancelled (simulates a long-running fetch).
	<-ctx.Done()

	close(ch)
	return nil, ctx.Err()
}

func (f *fakeFetcher) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// waitForReturn waits until the most recent fetch call returns.
func (f *fakeFetcher) waitForReturn(t *testing.T) {
	t.Helper()
	f.mu.Lock()
	ch := f.returned
	f.mu.Unlock()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for fetch to return")
	}
}

// fakeStorage satisfies CollectibleOwnershipStorage with no-ops.
// The zero value reports a timestamp of a previous successful fetch (non-initial load).
type fakeStorage struct {
	timestamp int64
}

func (s fakeStorage) GetOwnershipUpdateTimestamp(_ common.Address, _ walletCommon.ChainID) (int64, error) {
	return s.timestamp, nil
}

func (fakeStorage) Update(_ walletCommon.ChainID, _ common.Address, _ []thirdparty.CollectibleIDBalance, _ int64) ([]thirdparty.CollectibleUniqueID, []thirdparty.CollectibleUniqueID, []thirdparty.CollectibleUniqueID, error) {
	return nil, nil, nil, nil
}

func newTestLoader(t *testing.T, fetcher CollectibleOwnershipFetcher, storage CollectibleOwnershipStorage, params PeriodicalLoaderParams) *PeriodicalLoader {
	t.Helper()
	pl, _ := newTestLoaderWithPublisher(t, fetcher, storage, params)
	return pl
}

// newTestLoaderWithPublisher also returns the publisher the loader emits its
// events on, for tests that need to observe them.
func newTestLoaderWithPublisher(t *testing.T, fetcher CollectibleOwnershipFetcher, storage CollectibleOwnershipStorage, params PeriodicalLoaderParams) (*PeriodicalLoader, *pubsub.Publisher) {
	t.Helper()
	logger := zaptest.NewLogger(t)
	publisher := pubsub.NewPublisher()
	return NewPeriodicalLoader(
		walletCommon.ChainID(1),
		common.HexToAddress("0x1234"),
		fetcher,
		storage,
		publisher,
		params,
		logger,
	), publisher
}

// TestPeriodicalLoaderStartStop verifies that Start followed by Stop cleanly
// terminates the loader goroutines.
func TestPeriodicalLoaderStartStop(t *testing.T) {
	fetcher := newFakeFetcher()
	params := PeriodicalLoaderParams{
		LoadInterval: 1 * time.Second,
		LoaderParams: LoaderParams{FetchLimit: 50},
	}
	pl := newTestLoader(t, fetcher, fakeStorage{}, params)

	pl.Start(false, false)

	// Wait a bit for the goroutine to spin up and trigger the fetch.
	time.Sleep(100 * time.Millisecond)

	pl.Stop()

	// After Stop the state should eventually settle to idle or error (context cancelled).
	// The key assertion is that Stop() returns without hanging.
	require.NotEqual(t, LoaderStateDelayed, pl.GetState())
}

// TestPeriodicalLoaderRestartReleasesOldGoroutines verifies that calling
// Start a second time (which internally calls Stop then Start) does not leave
// the goroutines from the first Start hanging. This is the bug that the
// stopCh-capture fix addresses: without the fix, the first goroutine would
// read pl.stopCh (now nil after Stop) and block forever.
func TestPeriodicalLoaderRestartReleasesOldGoroutines(t *testing.T) {
	fetcher := newFakeFetcher()
	params := PeriodicalLoaderParams{
		LoadInterval: 1 * time.Second,
		LoaderParams: LoaderParams{FetchLimit: 50},
	}
	pl := newTestLoader(t, fetcher, fakeStorage{}, params)

	// First start — triggers an immediate load that blocks in the fetcher.
	pl.Start(false, false)
	// Wait for the first fetch to actually enter.
	<-fetcher.entered

	// Second start — internally calls Stop() then starts a new cycle.
	// Without the fix, the first goroutine would hang here because it reads
	// pl.stopCh which is now nil.
	pl.Start(false, false)

	// The first fetch should have been cancelled and returned.
	fetcher.waitForReturn(t)

	// Wait for the second fetch to enter.
	select {
	case <-fetcher.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("second fetch was never triggered — old goroutine may be hanging")
	}

	// Verify both starts triggered fetches.
	require.GreaterOrEqual(t, fetcher.CallCount(), 2, "expected at least 2 fetch calls after restart")

	// Clean up
	pl.Stop()
}

// TestLoadDelaySkippedOnInitialLoad verifies that the first ever load for an
// account+chain fetches right away instead of waiting out LoadDelay: there's
// nothing to debounce yet, and the delay is fully visible to the user.
func TestLoadDelaySkippedOnInitialLoad(t *testing.T) {
	fetcher := newFakeFetcher()
	params := PeriodicalLoaderParams{
		LoadInterval: 1 * time.Minute,
		LoaderParams: LoaderParams{
			LoadDelay:  30 * time.Second,
			FetchLimit: 50,
		},
	}
	pl := newTestLoader(t, fetcher, fakeStorage{timestamp: InvalidTimestamp}, params)

	pl.Start(false, false)

	select {
	case <-fetcher.entered:
	case <-time.After(5 * time.Second):
		pl.Stop()
		t.Fatal("initial load waited for LoadDelay before fetching")
	}

	require.Equal(t, LoaderStateUpdating, pl.GetState())

	pl.Stop()
	waitForLoadToEnd(t, pl)
}

// waitForLoadToEnd waits for an interrupted load to unwind, so that the loader
// goroutines don't outlive the test.
func waitForLoadToEnd(t *testing.T, pl *PeriodicalLoader) {
	t.Helper()
	require.Eventually(t, func() bool {
		state := pl.GetState()
		return state == LoaderStateIdle || state == LoaderStateError
	}, 5*time.Second, 10*time.Millisecond)
}

// TestLoadDelayReportedAsDelayed verifies that a subsequent load still waits out
// LoadDelay, and that it is reported as delayed (not as updating) while doing so.
func TestLoadDelayReportedAsDelayed(t *testing.T) {
	fetcher := newFakeFetcher()
	params := PeriodicalLoaderParams{
		LoadInterval: 1 * time.Minute,
		LoaderParams: LoaderParams{
			LoadDelay:  5 * time.Second,
			FetchLimit: 50,
		},
	}
	pl := newTestLoader(t, fetcher, fakeStorage{}, params)

	pl.Start(false, false)

	require.Eventually(t, func() bool {
		return pl.GetState() == LoaderStateDelayed
	}, 1*time.Second, 10*time.Millisecond)

	require.Equal(t, 0, fetcher.CallCount(), "no fetch should have been triggered during the load delay")

	pl.Stop()
	waitForLoadToEnd(t, pl)
}

// requireNoLoadError fails if a load error event is published within the given
// observation window.
func requireNoLoadError(t *testing.T, errCh chan EventOwnedCollectiblesLoadError, window time.Duration) {
	t.Helper()
	select {
	case event := <-errCh:
		require.FailNowf(t, "a cancelled load must not be reported as an error",
			"unexpected load error event published: %v", event.Error)
	case <-time.After(window):
	}
}

// TestCancelledLoadDuringDelayIsNotReportedAsError verifies that stopping the
// loader while a load is still waiting out LoadDelay ends that load silently:
// the pair simply isn't loading anymore, which must not reach the client as a
// failed load.
func TestCancelledLoadDuringDelayIsNotReportedAsError(t *testing.T) {
	fetcher := newFakeFetcher()
	params := PeriodicalLoaderParams{
		LoadInterval: 1 * time.Minute,
		LoaderParams: LoaderParams{
			LoadDelay:  5 * time.Second,
			FetchLimit: 50,
		},
	}
	pl, publisher := newTestLoaderWithPublisher(t, fetcher, fakeStorage{}, params)

	errCh, unsubFn := pubsub.Subscribe[EventOwnedCollectiblesLoadError](publisher, 10)
	defer unsubFn()

	pl.Start(false, false)

	require.Eventually(t, func() bool {
		return pl.GetState() == LoaderStateDelayed
	}, 1*time.Second, 10*time.Millisecond)

	// Stop while the load is still being debounced.
	pl.Stop()

	requireNoLoadError(t, errCh, 500*time.Millisecond)
	require.Equal(t, LoaderStateIdle, pl.GetState(), "a cancelled load must settle as idle, not as an error")
	require.Equal(t, 0, fetcher.CallCount(), "no fetch should have been triggered during the load delay")
}

// TestCancelledLoadDuringFetchIsNotReportedAsError verifies the same for a load
// that is already fetching when the loader is stopped.
func TestCancelledLoadDuringFetchIsNotReportedAsError(t *testing.T) {
	fetcher := newFakeFetcher()
	params := PeriodicalLoaderParams{
		LoadInterval: 1 * time.Minute,
		LoaderParams: LoaderParams{FetchLimit: 50},
	}
	pl, publisher := newTestLoaderWithPublisher(t, fetcher, fakeStorage{}, params)

	errCh, unsubFn := pubsub.Subscribe[EventOwnedCollectiblesLoadError](publisher, 10)
	defer unsubFn()

	pl.Start(false, false)

	select {
	case <-fetcher.entered:
	case <-time.After(5 * time.Second):
		pl.Stop()
		t.Fatal("timed out waiting for the fetch to start")
	}

	// Stop while the fetch is in flight.
	pl.Stop()
	fetcher.waitForReturn(t)

	requireNoLoadError(t, errCh, 500*time.Millisecond)
	require.Equal(t, LoaderStateIdle, pl.GetState(), "a cancelled load must settle as idle, not as an error")
}

// TestLoaderStateChangesForwardedToCaller verifies that a caller-supplied
// OnStateChanged callback still receives the load's state transitions. The
// PeriodicalLoader wraps that callback to track the state itself, and must chain
// to the caller's one instead of dropping it.
func TestLoaderStateChangesForwardedToCaller(t *testing.T) {
	fetcher := newFakeFetcher()

	var mu sync.Mutex
	var observedStates []LoaderState

	params := PeriodicalLoaderParams{
		LoadInterval: 1 * time.Minute,
		LoaderParams: LoaderParams{
			LoadDelay:  100 * time.Millisecond,
			FetchLimit: 50,
			OnStateChanged: func(state LoaderState) {
				mu.Lock()
				defer mu.Unlock()
				observedStates = append(observedStates, state)
			},
		},
	}
	pl := newTestLoader(t, fetcher, fakeStorage{}, params)

	pl.Start(false, false)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return slices.Contains(observedStates, LoaderStateDelayed) &&
			slices.Contains(observedStates, LoaderStateUpdating)
	}, 3*time.Second, 10*time.Millisecond, "the caller's OnStateChanged callback never saw the load progress")

	pl.Stop()
	waitForLoadToEnd(t, pl)
}

// TestPeriodicalLoaderStopDuringDelay verifies that Stop cancels the start
// delay and the loader returns to idle state.
func TestPeriodicalLoaderStopDuringDelay(t *testing.T) {
	fetcher := newFakeFetcher()
	params := PeriodicalLoaderParams{
		StartDelay:   5 * time.Second,
		LoadInterval: 1 * time.Second,
		LoaderParams: LoaderParams{FetchLimit: 50},
	}
	pl := newTestLoader(t, fetcher, fakeStorage{}, params)

	pl.Start(true, false)
	time.Sleep(100 * time.Millisecond)

	require.Equal(t, LoaderStateDelayed, pl.GetState())

	pl.Stop()

	// Give the goroutine time to react to the stop signal.
	time.Sleep(100 * time.Millisecond)

	require.Equal(t, LoaderStateIdle, pl.GetState())
	require.Equal(t, 0, fetcher.CallCount(), "no fetch should have been triggered during delay")
}
