package reliability

import (
	"testing"
	"time"

	"github.com/status-im/mvds/state"
	"github.com/stretchr/testify/require"
)

func TestPeerActivityReportOnlineDeduplicatesWithinTimeout(t *testing.T) {
	var activity peerActivity
	var peer state.PeerID
	now := time.Now()
	enqueues := 0
	enqueue := func() bool {
		enqueues++
		return true
	}

	require.Equal(t, "queued", activity.reportOnline(peer, now, enqueue))
	require.Equal(t, "already_online", activity.reportOnline(peer, now.Add(peerOnlineTimeout-time.Second), enqueue))
	require.Equal(t, 1, enqueues)
}

func TestPeerActivityOnlineStatusIsTrackedPerPeer(t *testing.T) {
	var activity peerActivity
	var firstPeer, secondPeer state.PeerID
	secondPeer[0] = 1
	now := time.Now()
	enqueues := 0
	enqueue := func() bool {
		enqueues++
		return true
	}

	require.Equal(t, "queued", activity.reportOnline(firstPeer, now, enqueue))
	require.Equal(t, "queued", activity.reportOnline(secondPeer, now, enqueue))
	require.Equal(t, "already_online", activity.reportOnline(firstPeer, now, enqueue))
	require.Equal(t, 2, enqueues)
}

func TestPeerActivityReportOnlineAllowsPeerAfterTimeout(t *testing.T) {
	var activity peerActivity
	var peer state.PeerID
	now := time.Now()
	enqueues := 0
	enqueue := func() bool {
		enqueues++
		return true
	}

	require.Equal(t, "queued", activity.reportOnline(peer, now, enqueue))
	require.Equal(t, "queued", activity.reportOnline(peer, now.Add(peerOnlineTimeout), enqueue))
	require.Equal(t, 2, enqueues)
}

func TestPeerActivityRecordRenewsOnlineTimeout(t *testing.T) {
	var activity peerActivity
	var peer state.PeerID
	now := time.Now()
	enqueues := 0
	enqueue := func() bool {
		enqueues++
		return true
	}

	require.Equal(t, "queued", activity.reportOnline(peer, now, enqueue))
	activity.record(peer, now.Add(peerOnlineTimeout-time.Second))
	require.Equal(t, "already_online", activity.reportOnline(peer, now.Add(peerOnlineTimeout+time.Second), enqueue))
	require.Equal(t, 1, enqueues)
}

func TestPeerActivityQueueFullDoesNotSuppressRetry(t *testing.T) {
	var activity peerActivity
	var peer state.PeerID
	now := time.Now()

	require.Equal(t, "queue_full", activity.reportOnline(peer, now, func() bool { return false }))
	enqueues := 0
	require.Equal(t, "queued", activity.reportOnline(peer, now.Add(time.Second), func() bool {
		enqueues++
		return true
	}))
	require.Equal(t, 1, enqueues)
}

func TestPeerActivityTrafficDoesNotSuppressRecoveryNotification(t *testing.T) {
	for _, tt := range []struct {
		name        string
		notifyFirst bool
		queueFull   bool
	}{
		{name: "traffic before first presence"},
		{name: "traffic after notified lease expires", notifyFirst: true},
		{name: "traffic after notification queue is full", queueFull: true},
		{name: "traffic after expired lease notification queue is full", notifyFirst: true, queueFull: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var activity peerActivity
			var peer state.PeerID
			now := time.Now()
			enqueues := 0
			enqueue := func() bool {
				enqueues++
				return true
			}
			if tt.notifyFirst {
				require.Equal(t, "queued", activity.reportOnline(peer, now, enqueue))
				now = now.Add(peerOnlineTimeout)
				// Prune another peer first so expiry must also work between pruning runs.
				var otherPeer state.PeerID
				otherPeer[0] = 1
				activity.record(otherPeer, now.Add(-time.Second))
			}
			if tt.queueFull {
				require.Equal(t, "queue_full", activity.reportOnline(peer, now, func() bool { return false }))
			}
			before := enqueues
			activity.record(peer, now)
			activity.record(peer, now.Add(time.Second))
			require.Equal(t, "queued", activity.reportOnline(peer, now.Add(2*time.Second), enqueue))
			require.Equal(t, before+1, enqueues)
			require.Equal(t, "already_online", activity.reportOnline(peer, now.Add(3*time.Second), enqueue))
			require.Equal(t, before+1, enqueues)
		})
	}
}

func TestPeerActivityPrunesExpiredPeers(t *testing.T) {
	var activity peerActivity
	var peer, nextPeer state.PeerID
	nextPeer[0] = 1
	now := time.Now()
	require.Equal(t, "queued", activity.reportOnline(peer, now, func() bool { return true }))

	require.Equal(t, "queued", activity.reportOnline(nextPeer, now.Add(peerOnlineTimeout+time.Nanosecond), func() bool { return true }))
	_, exists := activity.lastSeen[peer]
	require.False(t, exists)
}
