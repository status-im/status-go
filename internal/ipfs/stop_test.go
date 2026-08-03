package ipfs

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A well-formed ipfs-ns contenthash; the content is never actually fetched in
// these tests, the request only has to get past decoding.
const testContentHash = "e3010170122029f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f"

// waitForQueuedRequest blocks until Get has put its request on inputTaskChan,
// so the test controls the moment Stop is called rather than racing it.
func waitForQueuedRequest(t *testing.T, d *Downloader) {
	t.Helper()
	require.Eventually(t, func() bool {
		return len(d.inputTaskChan) > 0
	}, 2*time.Second, 5*time.Millisecond, "Get never queued its request")
}

// Regression: a request that is queued but not yet dispatched used to leave
// Stop blocked forever on wg.Wait(), wedging the whole logout path.
func TestStopReturnsWithUndispatchedRequest(t *testing.T) {
	d := NewDownloader(t.TempDir())

	// Pausing keeps the dispatcher from draining inputTaskChan, which makes the
	// otherwise sub-tick-length race window deterministic.
	require.NoError(t, d.Pause())

	getErr := make(chan error, 1)
	go func() {
		_, err := d.Get(testContentHash, false)
		getErr <- err
	}()
	waitForQueuedRequest(t, d)

	stopped := make(chan struct{})
	go func() {
		d.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return while a request was queued")
	}

	select {
	case err := <-getErr:
		require.Error(t, err, "Get should fail once the downloader is stopped")
	case <-time.After(2 * time.Second):
		t.Fatal("Get did not return after Stop")
	}
}

// The media server keeps serving requests while the node tears down, so Get is
// still called after Stop; it must fail rather than send on a closed channel.
func TestGetAfterStopFailsWithoutPanic(t *testing.T) {
	d := NewDownloader(t.TempDir())
	d.Stop()

	for i := 0; i < 50; i++ {
		_, err := d.Get(testContentHash, false)
		require.ErrorIs(t, err, ErrDownloaderStopped)
	}
}

// Regression: with rateLimiterChan full and no worker left to drain it, the
// dispatcher must still observe quit rather than park on its send forever.
func TestDispatcherUnblocksOnQuitWithFullRateLimiter(t *testing.T) {
	d := &Downloader{
		inputTaskChan:   make(chan taskRequest, 10),
		rateLimiterChan: make(chan taskRequest, 1),
		quit:            make(chan struct{}),
	}

	exited := make(chan struct{})
	go func() { d.taskDispatcher(); close(exited) }()

	// First fills rateLimiterChan, second parks the dispatcher on the send.
	d.inputTaskChan <- taskRequest{cid: "a", doneChan: make(chan taskResponse, 1)}
	d.inputTaskChan <- taskRequest{cid: "b", doneChan: make(chan taskResponse, 1)}
	time.Sleep(3 * time.Second / maxRequestsPerSecond)

	close(d.quit)

	select {
	case <-exited:
	case <-time.After(3 * time.Second):
		t.Fatal("dispatcher stayed blocked on a full rateLimiterChan after quit")
	}
}

// blockingRoundTripper parks the worker inside download() until the test
// releases it, so the test owns the moment Stop is called.
type blockingRoundTripper struct {
	inFlight chan struct{}
	release  chan struct{}
	entered  sync.Once
}

func newBlockingRoundTripper() *blockingRoundTripper {
	return &blockingRoundTripper{
		inFlight: make(chan struct{}),
		release:  make(chan struct{}),
	}
}

func (b *blockingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	b.entered.Do(func() { close(b.inFlight) })
	<-b.release
	return nil, errors.New("transport released")
}

// The node tears down its data directory right after Stop returns, so Stop must
// not hand back control while the worker is still inside download() writing into
// it. d.wg tracked only Get callers, never the worker or the dispatcher.
func TestStopWaitsForInFlightDownload(t *testing.T) {
	d := NewDownloader(t.TempDir())

	transport := newBlockingRoundTripper()
	d.client = &http.Client{Transport: transport}

	go func() { _, _ = d.Get(testContentHash, false) }()

	select {
	case <-transport.inFlight:
	case <-time.After(5 * time.Second):
		t.Fatal("worker never entered download")
	}

	stopped := make(chan struct{})
	go func() {
		d.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("Stop returned while the worker was still downloading")
	case <-time.After(300 * time.Millisecond):
	}

	close(transport.release)

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return after the download finished")
	}
}

// A select over a ready doneChan and a closed quit picks uniformly, so a result
// the worker already delivered used to be thrown away in favour of the shutdown
// error about half the time. Looped so a uniform pick cannot pass by luck.
func TestAwaitResultPrefersDeliveredResponse(t *testing.T) {
	d := &Downloader{quit: make(chan struct{})}
	close(d.quit)

	for i := 0; i < 100; i++ {
		doneChan := make(chan taskResponse, 1)
		doneChan <- taskResponse{response: []byte("payload")}

		resp, err := d.awaitResult(doneChan)
		require.NoError(t, err)
		require.Equal(t, []byte("payload"), resp)
	}
}

// Regression: Get registering on the WaitGroup must not race Stop's Wait,
// which panics with "WaitGroup misuse". Run this package with -race.
func TestConcurrentGetAndStop(t *testing.T) {
	for round := 0; round < 50; round++ {
		d := NewDownloader(t.TempDir())

		var callers sync.WaitGroup
		for i := 0; i < 8; i++ {
			callers.Add(1)
			go func() {
				defer callers.Done()
				_, _ = d.Get(testContentHash, false)
			}()
		}
		go d.Stop()
		callers.Wait()
	}
}

// Stop is exported; calling it twice must not panic on close of a closed channel.
func TestDoubleStopDoesNotPanic(t *testing.T) {
	d := NewDownloader(t.TempDir())
	d.Stop()
	d.Stop()
}
