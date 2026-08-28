package walletconnect

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCall_SingleFlightReconnect(t *testing.T) {
	fr := newFakeRelay(t, fakeRelayOpts{echoSubscribe: true})
	r := newTestRelayClient(t, fr)

	require.NoError(t, r.Connect())
	waitAccepted(t, fr, 1)

	fr.DropNow()
	// Give readLoop a moment to clear conn before hammering Subscribe.
	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	errs := make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := r.Subscribe(fmt.Sprintf("t-%d", i))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	waitAccepted(t, fr, 2)
	_ = r.Close()
}

func TestHeartbeat_DetectsDeadConnection(t *testing.T) {
	oldW, oldR, oldP := relayWriteDeadline, relayReadDeadline, relayPingInterval
	relayWriteDeadline = 2 * time.Second
	relayReadDeadline = 500 * time.Millisecond
	relayPingInterval = 200 * time.Millisecond
	defer func() {
		relayWriteDeadline, relayReadDeadline, relayPingInterval = oldW, oldR, oldP
	}()

	fr := newFakeRelay(t, fakeRelayOpts{echoSubscribe: true, silentOnPing: true})
	r := newTestRelayClient(t, fr)

	reconnected := make(chan struct{}, 8)
	r.SetReconnectedHandler(func() {
		reconnected <- struct{}{}
	})

	require.NoError(t, r.Connect())

	select {
	case <-reconnected:
	case <-time.After(3 * time.Second):
		t.Fatal("expected reconnect after dead heartbeat / read deadline")
	}
	_ = r.Close()
}

func TestClose_StopsHeartbeat(t *testing.T) {
	oldP := relayPingInterval
	relayPingInterval = 50 * time.Millisecond
	defer func() { relayPingInterval = oldP }()

	fr := newFakeRelay(t, fakeRelayOpts{echoSubscribe: true})
	r := newTestRelayClient(t, fr)

	require.NoError(t, r.Connect())

	done := make(chan struct{})
	go func() {
		_ = r.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Close hung — heartbeat/readLoop did not finish")
	}
}

func TestConnect_IsIdempotentWhenLive(t *testing.T) {
	fr := newFakeRelay(t, fakeRelayOpts{echoSubscribe: true})
	r := newTestRelayClient(t, fr)

	require.NoError(t, r.Connect())
	require.NoError(t, r.Connect())
	waitAccepted(t, fr, 1)
	_ = r.Close()
}

// The reconnected handler mirrors Client.onReconnected: it re-subscribes to the
// active topics over the freshly established connection. Subscribe is a
// request/response call whose reply can only be delivered by readLoop, so the
// handler must not be invoked on the readLoop goroutine itself.
func TestReconnectedHandler_ResubscribeDoesNotDeadlockReadLoop(t *testing.T) {
	fr := newFakeRelay(t, fakeRelayOpts{echoSubscribe: true})
	r := newTestRelayClient(t, fr)

	subscribed := make(chan error, 1)
	r.SetReconnectedHandler(func() {
		_, err := r.Subscribe("session-topic")
		subscribed <- err
	})

	require.NoError(t, r.Connect())
	waitAccepted(t, fr, 1)

	fr.DropNow()

	select {
	case err := <-subscribed:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("re-subscribe after reconnect never completed: readLoop cannot deliver the response while it is blocked inside the reconnected handler")
	}
	_ = r.Close()
}

// gorilla reports every non-101 upgrade response as the same opaque
// "websocket: bad handshake". The dial URL carries the auth JWT and the project
// id, so it must never be logged, but the status code tells an operator whether
// the relay rejected the credentials, the project, or rate-limited the client.
func TestDialRelay_ReportsRejectedUpgradeStatus(t *testing.T) {
	fr := newFakeRelay(t, fakeRelayOpts{rejectStatus: 401, rejectBody: "invalid jwt"})
	r := newTestRelayClient(t, fr)

	err := r.Connect()
	require.Error(t, err)
	require.Contains(t, err.Error(), "401", "dial error must carry the relay's HTTP status")
	require.NotContains(t, err.Error(), "auth=", "dial error must not leak the auth JWT")
	require.NotContains(t, err.Error(), "projectId", "dial error must not leak the project id")
}
