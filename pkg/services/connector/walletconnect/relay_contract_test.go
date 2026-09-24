package walletconnect

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/services/connector/walletconnect/relaytest"
)

// Contract tests: one per rule C1–C13 of RELAY_SPEC.md, black-box through the
// Relay API and relaytest.

// prompt bounds anything the contract calls "at once" or "promptly"; generous
// enough for -race on a loaded machine.
const prompt = time.Second

// setTunable overrides a package tunable for one test. Call it before
// newContractClient so the client is closed before the value is restored.
func setTunable[T any](t *testing.T, p *T, v T) {
	t.Helper()
	old := *p
	*p = v
	t.Cleanup(func() { *p = old })
}

// fastRedial makes redials quick so tests do not wait on the production backoff.
func fastRedial(t *testing.T) {
	t.Helper()
	setTunable(t, &relayReconnectBackoff, 5*time.Millisecond)
	setTunable(t, &relayReconnectMaxBackoff, 20*time.Millisecond)
}

func newContractClient(t *testing.T, mode relaytest.Mode) (*relaytest.Server, *RelayClient) {
	t.Helper()
	relay := relaytest.New(t, mode)
	r, err := NewRelayClient("test")
	require.NoError(t, err)
	WithRelayURL(relay.URL())(r)
	t.Cleanup(func() { _ = r.Close() })
	return relay, r
}

// async runs f on its own goroutine and returns a channel with its result.
func async(f func() error) <-chan error {
	ch := make(chan error, 1)
	go func() {
		defer panics.LogOnPanic()
		ch <- f()
	}()
	return ch
}

func subscribe(r *RelayClient) func() error {
	return func() error {
		_, err := r.Subscribe("session-topic")
		return err
	}
}

func requireResult(t *testing.T, ch <-chan error, within time.Duration, what string) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(within):
		t.Fatalf("%s did not return within %v", what, within)
		return nil
	}
}

// loseConnection drops the live connection with the relay switched to mode,
// and waits until the client has started redialing.
func loseConnection(t *testing.T, relay *relaytest.Server, mode relaytest.Mode) {
	t.Helper()
	held, rejected := relay.Held(), relay.Rejected()
	relay.SetMode(mode)
	relay.DropAll()
	require.Eventually(t, func() bool {
		return relay.Held() > held || relay.Rejected() > rejected
	}, 5*time.Second, 5*time.Millisecond, "client did not start redialing")
}

func TestRelay_C1_CallBeforeFirstConnectFailsAtOnce(t *testing.T) {
	calls := map[string]func(r *RelayClient) error{
		"Subscribe": func(r *RelayClient) error { return subscribe(r)() },
		"Publish":   func(r *RelayClient) error { return r.Publish("topic", "msg", 1108) },
		"FetchMessages": func(r *RelayClient) error {
			_, _, err := r.FetchMessages("topic")
			return err
		},
		"Unsubscribe": func(r *RelayClient) error { return r.Unsubscribe("topic", "sub-id") },
	}
	t.Run("never connected", func(t *testing.T) {
		relay, r := newContractClient(t, relaytest.Healthy)
		for name, call := range calls {
			start := time.Now()
			err := call(r)
			require.ErrorContains(t, err, "not connected", name)
			require.Less(t, time.Since(start), prompt, name)
		}
		require.Zero(t, relay.Accepted())
	})
	t.Run("first connect in flight", func(t *testing.T) {
		relay, r := newContractClient(t, relaytest.Blackhole)
		connected := async(r.Connect)
		require.Eventually(t, func() bool { return relay.Held() >= 1 }, 5*time.Second, 5*time.Millisecond)

		start := time.Now()
		_, err := r.Subscribe("topic")
		require.ErrorContains(t, err, "not connected")
		require.Less(t, time.Since(start), prompt)

		require.NoError(t, r.Close())
		require.Error(t, requireResult(t, connected, prompt, "Connect"))
	})
}

func TestRelay_C2_ConnectOnLiveConnectionDoesNotDial(t *testing.T) {
	relay, r := newContractClient(t, relaytest.Healthy)

	require.NoError(t, r.Connect())
	require.Eventually(t, func() bool { return relay.Accepted() == 1 }, 2*time.Second, 5*time.Millisecond)

	require.NoError(t, r.Connect())
	require.Never(t, func() bool { return relay.Accepted() > 1 }, 200*time.Millisecond, 10*time.Millisecond)
}

func TestRelay_C3_ConnectAfterCloseDoesNotDial(t *testing.T) {
	t.Run("after a live connection", func(t *testing.T) {
		relay, r := newContractClient(t, relaytest.Healthy)
		require.NoError(t, r.Connect())
		require.NoError(t, r.Close())
		accepted := relay.Accepted()

		err := r.Connect()
		require.ErrorIs(t, err, ErrRelayClosed)
		require.ErrorContains(t, err, "disconnect requested")
		require.Never(t, func() bool { return relay.Accepted() > accepted }, 200*time.Millisecond, 10*time.Millisecond)
	})
	t.Run("never connected", func(t *testing.T) {
		relay, r := newContractClient(t, relaytest.Healthy)
		require.NoError(t, r.Close())

		require.ErrorIs(t, r.Connect(), ErrRelayClosed)
		require.Never(t, func() bool { return relay.Accepted() > 0 }, 200*time.Millisecond, 10*time.Millisecond)
	})
}

func TestRelay_C4_FailedFirstConnectLeavesClientUsable(t *testing.T) {
	fastRedial(t)
	relay, r := newContractClient(t, relaytest.Reject)

	err := r.Connect()
	require.ErrorContains(t, err, "503")
	require.Equal(t, int32(1), relay.Rejected())
	require.Never(t, func() bool { return relay.Rejected() > 1 }, 200*time.Millisecond, 10*time.Millisecond,
		"a failed first Connect must not redial in the background")
	_, err = r.Subscribe("topic")
	require.ErrorContains(t, err, "not connected")

	relay.SetMode(relaytest.Healthy)
	require.NoError(t, r.Connect())
	require.NoError(t, subscribe(r)())
	require.Equal(t, int32(1), relay.Accepted())
}

func TestRelay_C5_RedialsUntilTheRelayComesBack(t *testing.T) {
	fastRedial(t)
	relay, r := newContractClient(t, relaytest.Healthy)

	require.NoError(t, r.Connect())
	relay.SetMode(relaytest.Reject)
	relay.DropAll()
	require.Eventually(t, func() bool { return relay.Rejected() >= 15 }, 5*time.Second, 5*time.Millisecond,
		"client stopped redialing")

	relay.SetMode(relaytest.Healthy)
	require.Eventually(t, func() bool { return subscribe(r)() == nil }, 10*time.Second, 50*time.Millisecond,
		"client never reconnected after the relay came back")
}

func TestRelay_C6_CallWaitsAtMostTheBudget(t *testing.T) {
	for name, mode := range map[string]relaytest.Mode{
		"redial hangs in the handshake": relaytest.Blackhole,
		"redial is rejected":            relaytest.Reject,
	} {
		t.Run(name, func(t *testing.T) {
			const budget = 200 * time.Millisecond
			setTunable(t, &relayReconnectWait, budget)
			fastRedial(t)
			relay, r := newContractClient(t, relaytest.Healthy)
			require.NoError(t, r.Connect())
			loseConnection(t, relay, mode)

			start := time.Now()
			err := requireResult(t, async(func() error { return r.Publish("topic", "payload", 1108) }),
				budget+prompt, "Publish")
			elapsed := time.Since(start)

			require.ErrorContains(t, err, "reconnect in progress")
			require.GreaterOrEqual(t, elapsed, budget)
			require.Less(t, elapsed, budget+prompt)
		})
	}
}

func TestRelay_C7_ParkedCallIsServedOnTheNewConnection(t *testing.T) {
	setTunable(t, &relayReconnectWait, 5*time.Second)
	fastRedial(t)
	relay, r := newContractClient(t, relaytest.Healthy)
	require.NoError(t, r.Connect())
	loseConnection(t, relay, relaytest.Reject)

	subscribed := async(subscribe(r))
	select {
	case err := <-subscribed:
		t.Fatalf("call returned while the relay was unreachable: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	relay.SetMode(relaytest.Healthy)
	require.NoError(t, requireResult(t, subscribed, 5*time.Second, "parked Subscribe"))
	require.Equal(t, int32(1), relay.Subscribes())
	require.Equal(t, int32(2), relay.Accepted())
}

func TestRelay_C8_ReconnectedHandlerRunsOncePerRedial(t *testing.T) {
	fastRedial(t)
	relay, r := newContractClient(t, relaytest.Healthy)
	var runs atomic.Int32
	r.SetReconnectedHandler(func() { runs.Add(1) })

	require.NoError(t, r.Connect())
	require.Never(t, func() bool { return runs.Load() > 0 }, 200*time.Millisecond, 10*time.Millisecond,
		"handler ran after the first Connect")

	for want := int32(1); want <= 2; want++ {
		relay.DropAll()
		require.Eventually(t, func() bool { return runs.Load() == want }, 5*time.Second, 5*time.Millisecond)
		require.Never(t, func() bool { return runs.Load() > want }, 200*time.Millisecond, 10*time.Millisecond,
			"handler ran more than once for one redial")
	}
}

// The handler mirrors Client.onReconnected: it re-subscribes over the new
// connection, which needs the reader to deliver the response, so the handler
// must run on its own goroutine.
func TestRelay_C8_ReconnectedHandlerMayMakeCalls(t *testing.T) {
	fastRedial(t)
	relay, r := newContractClient(t, relaytest.Healthy)
	subscribed := make(chan error, 1)
	r.SetReconnectedHandler(func() { subscribed <- subscribe(r)() })

	require.NoError(t, r.Connect())
	relay.DropAll()

	require.NoError(t, requireResult(t, subscribed, 5*time.Second, "re-subscribe from the reconnected handler"))
}

func TestRelay_C9_CloseAbortsAHandshakeTheRelayNeverAnswers(t *testing.T) {
	requireCloseIsPrompt := func(t *testing.T, relay *relaytest.Server, r *RelayClient) {
		t.Helper()
		require.NoError(t, requireResult(t, async(r.Close), prompt, "Close"))
		require.Eventually(t, func() bool { return relay.Aborted() >= 1 }, prompt, 5*time.Millisecond,
			"the held handshake socket was not closed")
	}
	t.Run("first connect", func(t *testing.T) {
		relay, r := newContractClient(t, relaytest.Blackhole)
		connected := async(r.Connect)
		require.Eventually(t, func() bool { return relay.Held() >= 1 }, 5*time.Second, 5*time.Millisecond)

		requireCloseIsPrompt(t, relay, r)
		require.Error(t, requireResult(t, connected, prompt, "Connect"))
	})
	t.Run("redial", func(t *testing.T) {
		fastRedial(t)
		relay, r := newContractClient(t, relaytest.Healthy)
		require.NoError(t, r.Connect())
		loseConnection(t, relay, relaytest.Blackhole)

		requireCloseIsPrompt(t, relay, r)
	})
}

func TestRelay_C10_CloseFailsEveryCallAndNeverDialsAgain(t *testing.T) {
	requireShuttingDown := func(t *testing.T, ch <-chan error) {
		t.Helper()
		require.ErrorContains(t, requireResult(t, ch, prompt, "call"), "shutting down")
	}
	t.Run("pending call", func(t *testing.T) {
		relay, r := newContractClient(t, relaytest.Mute)
		require.NoError(t, r.Connect())
		pending := async(subscribe(r))
		require.Eventually(t, func() bool { return relay.Requests() >= 1 }, 2*time.Second, 5*time.Millisecond)

		require.NoError(t, r.Close())
		requireShuttingDown(t, pending)
	})
	t.Run("parked call", func(t *testing.T) {
		setTunable(t, &relayReconnectWait, 5*time.Second)
		fastRedial(t)
		relay, r := newContractClient(t, relaytest.Healthy)
		require.NoError(t, r.Connect())
		loseConnection(t, relay, relaytest.Reject)
		parked := async(subscribe(r))
		time.Sleep(50 * time.Millisecond)

		require.NoError(t, r.Close())
		requireShuttingDown(t, parked)
	})
	t.Run("call after close", func(t *testing.T) {
		_, r := newContractClient(t, relaytest.Healthy)
		require.NoError(t, r.Connect())
		require.NoError(t, r.Close())

		requireShuttingDown(t, async(subscribe(r)))
	})
	t.Run("no socket after close", func(t *testing.T) {
		fastRedial(t)
		relay, r := newContractClient(t, relaytest.Healthy)
		require.NoError(t, r.Connect())
		loseConnection(t, relay, relaytest.Reject)

		require.NoError(t, r.Close())
		rejected := relay.Rejected()
		relay.SetMode(relaytest.Healthy)
		require.Never(t, func() bool { return relay.Accepted() > 1 || relay.Rejected() > rejected },
			300*time.Millisecond, 10*time.Millisecond, "client dialed after Close")
	})
	t.Run("idempotent", func(t *testing.T) {
		_, r := newContractClient(t, relaytest.Healthy)
		require.NoError(t, r.Connect())
		require.NoError(t, r.Close())
		require.NoError(t, requireResult(t, async(r.Close), prompt, "second Close"))
	})
}

func TestRelay_C11_UnansweredCallTimesOut(t *testing.T) {
	require.Equal(t, 30*time.Second, relayCallTimeout, "production call timeout")

	const timeout = 200 * time.Millisecond
	setTunable(t, &relayCallTimeout, timeout)
	relay, r := newContractClient(t, relaytest.Mute)
	require.NoError(t, r.Connect())

	start := time.Now()
	err := requireResult(t, async(subscribe(r)), timeout+prompt, "Subscribe")
	elapsed := time.Since(start)

	require.ErrorContains(t, err, "relay call timeout")
	require.GreaterOrEqual(t, elapsed, timeout)
	require.Equal(t, int32(1), relay.Requests())
}

// A relay that stops reading makes the write block until its deadline; the
// payload is far larger than the loopback socket buffers.
func TestRelay_C12_FailedWriteIsRetriedOnceOnTheNextConnection(t *testing.T) {
	setTunable(t, &relayWriteDeadline, time.Second)
	setTunable(t, &relayReconnectWait, 10*time.Second)
	setTunable(t, &relayCallTimeout, 10*time.Second)
	fastRedial(t)
	relay, r := newContractClient(t, relaytest.Stall)
	require.NoError(t, r.Connect())
	relay.SetMode(relaytest.Healthy)

	payload := strings.Repeat("x", 16<<20)
	err := requireResult(t, async(func() error { return r.Publish("topic", payload, 1108) }), 15*time.Second, "Publish")

	require.NoError(t, err)
	require.Equal(t, int32(2), relay.Accepted())
	require.Equal(t, int32(1), relay.Requests(), "the request must be written exactly once more")
}

func TestRelay_C13_MissedHeartbeatMarksTheConnectionLost(t *testing.T) {
	setTunable(t, &relayReadDeadline, 300*time.Millisecond)
	setTunable(t, &relayPingInterval, 100*time.Millisecond)
	fastRedial(t)
	relay, r := newContractClient(t, relaytest.Mute)
	reconnected := make(chan struct{}, 8)
	r.SetReconnectedHandler(func() { reconnected <- struct{}{} })

	require.NoError(t, r.Connect())

	select {
	case <-reconnected:
	case <-time.After(3 * time.Second):
		t.Fatal("expected a redial after the heartbeat went unanswered")
	}
	require.GreaterOrEqual(t, relay.Accepted(), int32(2))
}
