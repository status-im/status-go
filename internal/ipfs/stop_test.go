package ipfs

import (
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
