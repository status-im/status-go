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
	// Give the client a moment to notice the loss before hammering Subscribe.
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
		t.Fatal("Close hung — heartbeat/reader did not finish")
	}
}
