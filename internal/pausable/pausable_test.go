package pausable

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// drainPauseSubscription reads every value currently queued on sub.C() without blocking.
// PauseBroadcaster delivers an initial snapshot then transition events on the same channel;
// with a buffer of 1, a transition can be dropped while the buffer still holds the snapshot
// (see pkg/pubsub.TypePublisher Send). Subscribers that care about correctness should
// treat the last drained value as the effective pause flag after catching up.
func drainPauseSubscription(sub Subscription) []bool {
	ch := sub.C()
	var vals []bool
	for {
		select {
		case v := <-ch:
			vals = append(vals, v)
		default:
			return vals
		}
	}
}

// These tests guard against losing a MarkPaused/MarkResumed between Subscribe and the first
// read: the initial IsPaused() snapshot must not fill the per-subscriber buffer such that the
// following Publish is dropped while PausableState() has already changed.

func TestPauseBroadcaster_Subscribe_pauseBeforeFirstRead_lastQueuedMatchesState(t *testing.T) {
	var pb PauseBroadcaster
	pb.MarkStarted()
	sub := pb.Subscribe()
	pb.MarkPaused()
	require.Equal(t, ServiceStatePaused, pb.PausableState())

	vals := drainPauseSubscription(sub)
	require.NotEmpty(t, vals, "expected at least the initial snapshot on the subscription channel")
	last := vals[len(vals)-1]
	require.True(t, last, "after MarkPaused before any read, latest queued value should be paused (true); got %v for sequence %v", last, vals)
	sub.Unsubscribe()
}

func TestPauseBroadcaster_Subscribe_resumeBeforeFirstRead_lastQueuedMatchesState(t *testing.T) {
	var pb PauseBroadcaster
	pb.MarkStarted()
	pb.MarkPaused()
	sub := pb.Subscribe()
	pb.MarkResumed()
	require.Equal(t, ServiceStateRunning, pb.PausableState())

	vals := drainPauseSubscription(sub)
	require.NotEmpty(t, vals, "expected at least the initial snapshot on the subscription channel")
	last := vals[len(vals)-1]
	require.False(t, last, "after MarkResumed before any read, latest queued value should be running (false); got %v for sequence %v", last, vals)
	sub.Unsubscribe()
}
