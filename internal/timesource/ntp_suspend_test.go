package timesource

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/require"
)

// readBudget is how long reading the current time may take on the caller's thread.
const readBudget = 250 * time.Millisecond

// After the device sleeps, the wall clock is ahead of the monotonic clock by the
// time asleep. Reading the time must not wait for NTP servers that do not answer.
func TestGetCurrentTimeDoesNotWaitForNTPAfterSuspend(t *testing.T) {
	for _, asleep := range []time.Duration{0, 3 * time.Hour} {
		t.Run("asleep "+asleep.String(), func(t *testing.T) {
			readTimeAfterSleep(t, asleep)
		})
	}
}

func readTimeAfterSleep(t *testing.T, asleep time.Duration) {
	stalled := make(chan struct{})
	t.Cleanup(func() { close(stalled) })
	var queries atomic.Int32

	source := &ntpTimeSource{
		servers:         mockedServers,
		allowedFailures: DefaultMaxAllowedFailures,
		timeQuery: func(string, ntp.QueryOptions) (*ntp.Response, error) {
			queries.Add(1)
			<-stalled
			return nil, context.DeadlineExceeded
		},
		now:           func() time.Time { return time.Now().Add(asleep) },
		started:       true,
		lastMonotonic: time.Now(),
	}

	done := make(chan struct{})
	go func() {
		source.GetCurrentTimeInMillis()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(readBudget):
		require.Positive(t, queries.Load(), "blocked before reaching NTP")
		t.Fatalf("GetCurrentTimeInMillis blocked for over %v waiting on %d NTP queries", readBudget, queries.Load())
	}
}

// Resolving a server name is not covered by the NTP query timeout, so the
// offset computation must stop waiting on its own.
func TestComputeOffsetStopsWaitingForNameResolution(t *testing.T) {
	const deadline = 200 * time.Millisecond

	var (
		dials   atomic.Int32
		queries sync.WaitGroup
	)
	release := make(chan struct{})
	saved := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dials.Add(1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-release:
				return nil, context.Canceled
			}
		},
	}
	t.Cleanup(func() {
		close(release)
		queries.Wait()
		net.DefaultResolver = saved
	})

	query := func(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
		defer queries.Done()
		return ntp.QueryWithOptions(host, opt)
	}
	queries.Add(1)
	done := make(chan error, 1)
	go func() {
		_, err := computeOffset(query, []string{"pool.ntp.org"}, 0, deadline)
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorContains(t, err, errQueryDeadline.Error())
	case <-time.After(5 * deadline):
		require.Positive(t, dials.Load(), "blocked before reaching the resolver")
		t.Fatalf("computeOffset with a %v deadline still running after %v, stuck in name resolution", deadline, 5*deadline)
	}
}

// A failed sync after a clock jump must be retried at the fast period,
// not an hour later at the slow one.
func TestFailedSyncAfterClockJumpRetriesAtFastPeriod(t *testing.T) {
	var queries atomic.Int32
	source := &ntpTimeSource{
		servers:         []string{"ntp1"},
		allowedFailures: 0,
		timeQuery: func(string, ntp.QueryOptions) (*ntp.Response, error) {
			queries.Add(1)
			return nil, errors.New("offline")
		},
		fastNTPSyncPeriod: 20 * time.Millisecond,
		slowNTPSyncPeriod: time.Hour,
		now:               func() time.Time { return time.Now().Add(time.Hour) },
		lastMonotonic:     time.Now(),
		fastRetry:         make(chan struct{}, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// The last sync succeeded, so the periodic loop is on the slow period.
	source.runPeriodically(ctx, source.updateOffset, true)

	source.Now()

	require.Eventually(t, func() bool { return queries.Load() >= 2 }, time.Second, 5*time.Millisecond,
		"sync was not retried after the clock-jump sync failed")
}

// The wall clock drifts against the monotonic clock while the OS disciplines it.
// A successful sync must move the reference, so drift accumulated before it is
// not mistaken for a clock change later.
func TestSuccessfulSyncResetsClockJumpReference(t *testing.T) {
	start := time.Now()
	syncTime := start.Add(800 * time.Millisecond)
	source := &ntpTimeSource{
		servers:         []string{"ntp1"},
		allowedFailures: 0,
		timeQuery: func(string, ntp.QueryOptions) (*ntp.Response, error) {
			return &ntp.Response{Stratum: 1}, nil
		},
		now:           func() time.Time { return syncTime },
		lastMonotonic: start,
	}

	require.NoError(t, source.updateOffset())
	require.Equal(t, syncTime, source.lastMonotonic, "reference was not moved to the sync time")
}
