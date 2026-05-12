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
