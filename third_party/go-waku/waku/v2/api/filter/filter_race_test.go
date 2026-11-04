package filter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"go.uber.org/zap"
)

// TestSub_CleanupDoesNotDeadlockWhenSubChannelStaysOpen verifies that cleanup()
// returns even when subDetails.C is never closed by Unsubscribe. This is the
// real-world failure mode during node-stop transitions, where
// UnsubscribeWithSubscription early-returns from ErrOnNotRunning() and does NOT
// call sub.Close(), leaving subDetails.C open. The forwarder goroutine MUST
// react to apiSub.ctx.Done() — not depend on subDetails.C closing — so
// multiplexWG.Wait() in cleanup() can complete and the filter teardown isn't
// deadlocked. The test asserts cleanup() returns within a short timeout.
func TestSub_CleanupDoesNotDeadlockWhenSubChannelStaysOpen(t *testing.T) {
	apiSub := &Sub{
		DataCh: make(chan *protocol.Envelope, 16),
		subs:   make(subscription.SubscriptionSet),
		log:    zap.NewNop(),
	}
	apiSub.ctx, apiSub.cancel = context.WithCancel(context.Background())

	sd := &subscription.SubscriptionDetails{
		ID:      "test-deadlock",
		C:       make(chan *protocol.Envelope), // intentionally NEVER closed
		Closing: make(chan bool, 1),
	}

	apiSub.multiplex([]*subscription.SubscriptionDetails{sd})

	// Same HACK as the other test — avoid the nil wf path in cleanup().
	apiSub.subs = make(subscription.SubscriptionSet)

	// Let the forwarder goroutine reach its blocking receive on sd.C.
	time.Sleep(10 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		apiSub.cancel()
		apiSub.cleanup()
		close(done)
	}()

	select {
	case <-done:
		// cleanup() returned — the forwarder respected ctx.Done() and exited,
		// multiplexWG.Wait() unblocked.
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup() did not return within 2s — likely deadlocked on " +
			"an uncancelable receive from subDetails.C")
	}
}

// TestSub_CleanupRaceWithMultiplex exercises the race between Sub.cleanup() closing
// apiSub.DataCh and the per-subscription forwarder goroutines spawned by Sub.multiplex()
// executing apiSub.DataCh <- env.
//
// Without the WaitGroup + ctx-aware select fix, the forwarder goroutine is blocked on
// a send to an unbuffered DataCh, cleanup() proceeds to close(DataCh), the send wakes
// on a closed channel, and utils.LogOnPanic re-panics — killing the test binary.
//
// With the fix, the forwarder's select exits via apiSub.ctx.Done(), cleanup()'s
// multiplexWG.Wait() unblocks, and close(DataCh) is safe.
//
// The test does not use //go:build !race (unlike filter_test.go in this package)
// because the failure mode here is a runtime panic, not a data race the detector
// surfaces. It is fine — and good practice — to also run it under -race.
func TestSub_CleanupRaceWithMultiplex(t *testing.T) {
	for i := 0; i < 50; i++ {
		runCleanupRaceIteration(t)
	}
}

func runCleanupRaceIteration(t *testing.T) {
	t.Helper()

	apiSub := &Sub{
		// Unbuffered DataCh with no consumer forces the forwarder to block on send,
		// putting it in the exact state the production race depends on.
		DataCh: make(chan *protocol.Envelope),
		subs:   make(subscription.SubscriptionSet),
		log:    zap.NewNop(),
	}
	apiSub.ctx, apiSub.cancel = context.WithCancel(context.Background())

	sd := &subscription.SubscriptionDetails{
		ID:      "test-sub",
		C:       make(chan *protocol.Envelope, 100),
		Closing: make(chan bool, 1),
	}

	// Pre-fill the per-sub channel so the forwarder has envelopes ready to pump.
	for j := 0; j < 50; j++ {
		sd.C <- &protocol.Envelope{}
	}

	// Spawn the production multiplex code under test.
	apiSub.multiplex([]*subscription.SubscriptionDetails{sd})

	// HACK: clear apiSub.subs so cleanup() does not invoke
	// apiSub.wf.UnsubscribeWithSubscription on a nil WakuFilterLightNode. This test
	// targets the close-vs-send race, not the unsubscribe RPC. The forwarder
	// goroutines spawned by multiplex() still hold their own references to sd, so
	// they keep pumping after this clear.
	apiSub.subs = make(subscription.SubscriptionSet)

	// Yield long enough for the forwarder goroutine to begin and block on the
	// send to apiSub.DataCh. 10 ms is comfortably above the typical scheduling
	// latency even on heavily loaded CI machines.
	time.Sleep(10 * time.Millisecond)

	// Reproduce what subscriptionLoop does on apiSub.ctx.Done().
	apiSub.cancel()
	apiSub.cleanup()

	// If the unpatched bug is present, the forwarder goroutine has already
	// panicked above and killed the test binary via utils.LogOnPanic's re-panic.
	// Reaching this line means the close(DataCh) raced cleanly.
}

// TestFilterManager_SubscribeFilter_DoesNotDeadlockWhenQueueFull reproduces the
// capacity deadlock observed in production: SubscribeFilter holds mgr.Lock() and
// performs `mgr.waitingToSubQueue <- afilter` at filter_manager.go:146 while the
// node is offline. The queue is bounded (cap 100). The only drainer
// (checkAndProcessQueue) is itself invoked inside mgr.Lock() — so once the
// queue fills, the next send blocks forever, holding the lock, and the entire
// FilterManager (filter receive, messenger, notifications) freezes.
//
// Setup details:
//   - Each call uses the same PubsubTopic and 50 ContentTopics. The second and
//     every subsequent call sums to 50+50=100 > filterSubBatchSize(=90), which
//     triggers the offline branch at filter_manager.go:143-147 — pushing one
//     batch into waitingToSubQueue per call (after the first).
//   - ~102 calls fill the cap-100 queue; the 103rd will block indefinitely
//     under the unpatched code. We run 200 calls to be comfortably past the
//     threshold even if filterSubBatchSize changes.
//
// Pre-fix: test hangs and fails at the 5s deadline.
// Post-fix (slice-backed queue): test completes in milliseconds.
func TestFilterManager_SubscribeFilter_DoesNotDeadlockWhenQueueFull(t *testing.T) {
	mgr := &FilterManager{
		logger:                zap.NewNop(),
		onlineChecker:         onlinechecker.NewDefaultOnlineChecker(false).(*onlinechecker.DefaultOnlineChecker),
		incompleteFilterBatch: make(map[string]filterConfig),
		filterConfigs:         make(appFilterMap),
		filterSubscriptions:   make(map[string]SubDetails),
		// waitingToSubQueue is now a slice; zero-value nil is the empty queue.
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			cf := protocol.ContentFilter{
				PubsubTopic:   "/waku/2/rs/16/32",
				ContentTopics: makeNContentTopics(50, i),
			}
			mgr.SubscribeFilter(fmt.Sprintf("filter-%d", i), cf)
		}
	}()

	select {
	case <-done:
		// Pass — no deadlock under offline overflow.
	case <-time.After(5 * time.Second):
		t.Fatal("FilterManager.SubscribeFilter deadlocked when waitingToSubQueue overflowed; " +
			"a producer holding mgr.Lock() blocked on a bounded channel whose only drainer " +
			"also needs mgr.Lock(). Fix: replace the bounded chan with a mutex-guarded slice.")
	}
}

// makeNContentTopics returns a ContentTopicSet of n unique topics, seeded so
// repeated calls with different seeds yield non-overlapping topic sets.
func makeNContentTopics(n, seed int) protocol.ContentTopicSet {
	topics := make([]string, n)
	for j := 0; j < n; j++ {
		topics[j] = fmt.Sprintf("/test/%d-%d/proto", seed, j)
	}
	return protocol.NewContentTopicSet(topics...)
}

// TestShouldHonourRateLimitBackoff covers the pure predicate that gates retry
// triggers in subscriptionLoop once a peer has responded with HTTP 429. While
// in the backoff window the subscriptionLoop must stop pushing to apiSub.closing
// and stop calling checkAndResubscribe — otherwise it would keep slamming peers
// that explicitly asked us to back off.
func TestShouldHonourRateLimitBackoff(t *testing.T) {
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name             string
		rateLimitedUntil time.Time
		now              time.Time
		expect           bool
	}{
		{
			name:             "zero rateLimitedUntil (never set) — false",
			rateLimitedUntil: time.Time{},
			now:              now,
			expect:           false,
		},
		{
			name:             "now == rateLimitedUntil exactly — false (window has elapsed)",
			rateLimitedUntil: now,
			now:              now,
			expect:           false,
		},
		{
			name:             "now == rateLimitedUntil - 1ns — true (still inside window)",
			rateLimitedUntil: now.Add(1 * time.Nanosecond),
			now:              now,
			expect:           true,
		},
		{
			name:             "now within 30s of a 60s backoff — true",
			rateLimitedUntil: now.Add(30 * time.Second),
			now:              now,
			expect:           true,
		},
		{
			name:             "now after the backoff window — false",
			rateLimitedUntil: now.Add(-1 * time.Second),
			now:              now,
			expect:           false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expect, shouldHonourRateLimitBackoff(tc.rateLimitedUntil, tc.now))
		})
	}
}

// TestSub_BackoffAfterRateLimit_GateLogic exercises the Sub fields directly:
// when rateLimitedUntil is set in the future, subscriptionLoop's logic gates
// must reject retry triggers. This is a unit-level check against the Sub
// struct's contract, not a full subscribe-flow integration test (which would
// require stubbing *filter.WakuFilterLightNode, currently a concrete struct).
func TestSub_BackoffAfterRateLimit_GateLogic(t *testing.T) {
	apiSub := &Sub{log: zap.NewNop()}
	apiSub.ctx, apiSub.cancel = context.WithCancel(context.Background())
	defer apiSub.cancel()

	// Before any rate-limit, the gate must allow retries.
	require.False(t, shouldHonourRateLimitBackoff(apiSub.rateLimitedUntil, time.Now()),
		"with zero rateLimitedUntil, backoff gate must allow retries")

	// Simulate a 429: subscribe() sets rateLimitedUntil to now+filterRateLimitBackoff.
	apiSub.rateLimitedUntil = time.Now().Add(filterRateLimitBackoff)
	require.True(t, shouldHonourRateLimitBackoff(apiSub.rateLimitedUntil, time.Now()),
		"after setting rateLimitedUntil = now + filterRateLimitBackoff, gate must suppress retries")

	// Simulate the success path clearing the backoff.
	apiSub.rateLimitedUntil = time.Time{}
	require.False(t, shouldHonourRateLimitBackoff(apiSub.rateLimitedUntil, time.Now()),
		"after subscribe success clears rateLimitedUntil, gate must allow retries again")
}
