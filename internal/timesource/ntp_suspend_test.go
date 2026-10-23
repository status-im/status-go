package timesource

import (
	"context"
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
