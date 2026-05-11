package walletconnect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// fakeRelay is an in-process WalletConnect IRN-like WebSocket server for reconnect tests.
type fakeRelay struct {
	Server *httptest.Server

	mu          sync.Mutex
	currentConn *websocket.Conn

	accepted      int32
	forceDropAt   int32 // after N-th accepted connection, close immediately after upgrade
	echoSubscribe bool
	silentOnPing  bool
}

func newFakeRelay(t *testing.T, forceDropAt int32, echoSubscribe, silentOnPing bool) *fakeRelay {
	t.Helper()
	fr := &fakeRelay{
		forceDropAt:   forceDropAt,
		echoSubscribe: echoSubscribe,
		silentOnPing:  silentOnPing,
	}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		n := atomic.AddInt32(&fr.accepted, 1)
		if fr.forceDropAt > 0 && n == fr.forceDropAt {
			_ = conn.Close()
			return
		}
		if fr.silentOnPing {
			conn.SetPingHandler(func(string) error { return nil })
		}
		fr.mu.Lock()
		fr.currentConn = conn
		fr.mu.Unlock()
		defer func() {
			fr.mu.Lock()
			if fr.currentConn == conn {
				fr.currentConn = nil
			}
			fr.mu.Unlock()
		}()

		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			_ = mt
			if !fr.echoSubscribe {
				continue
			}
			var reqMsg struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
			}
			if err := json.Unmarshal(message, &reqMsg); err != nil {
				continue
			}
			if reqMsg.Method != "irn_subscribe" {
				continue
			}
			resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":"sub-id"}`, string(reqMsg.ID))
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resp)); err != nil {
				return
			}
		}
	})
	fr.Server = httptest.NewServer(mux)
	t.Cleanup(fr.Server.Close)
	return fr
}

func (fr *fakeRelay) wsURL() string {
	u := fr.Server.URL
	u = strings.Replace(u, "http://", "ws://", 1)
	u = strings.Replace(u, "https://", "wss://", 1)
	return u
}

func (fr *fakeRelay) DropNow() {
	fr.mu.Lock()
	c := fr.currentConn
	fr.mu.Unlock()
	if c != nil {
		_ = c.Close()
	}
}

func newTestRelayClient(t *testing.T, fr *fakeRelay) *RelayClient {
	t.Helper()
	r, err := NewRelayClient("test")
	require.NoError(t, err)
	r.url = fr.wsURL()
	return r
}

func TestCall_RetriesOnBrokenWrite(t *testing.T) {
	fr := newFakeRelay(t, 1, true, false)
	r := newTestRelayClient(t, fr)

	require.NoError(t, r.Connect())
	// Let readLoop observe EOF after forceDrop before Subscribe races a buffered write on a dead conn.
	time.Sleep(50 * time.Millisecond)
	_, err := r.Subscribe("topic-a")
	require.NoError(t, err)
	require.EqualValues(t, 2, atomic.LoadInt32(&fr.accepted))
	_ = r.Close()
}

func TestCall_SingleFlightReconnect(t *testing.T) {
	fr := newFakeRelay(t, 0, true, false)
	r := newTestRelayClient(t, fr)

	require.NoError(t, r.Connect())
	require.EqualValues(t, 1, atomic.LoadInt32(&fr.accepted))

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

	require.EqualValues(t, 2, atomic.LoadInt32(&fr.accepted))
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

	fr := newFakeRelay(t, 0, true, true)
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

func TestReadLoop_UsesSingleFlight(t *testing.T) {
	fr := newFakeRelay(t, 0, true, false)
	r := newTestRelayClient(t, fr)

	require.NoError(t, r.Connect())
	require.EqualValues(t, 1, atomic.LoadInt32(&fr.accepted))

	fr.DropNow()
	time.Sleep(50 * time.Millisecond)

	_, err := r.Subscribe("t1")
	require.NoError(t, err)
	require.EqualValues(t, 2, atomic.LoadInt32(&fr.accepted))
	_ = r.Close()
}

func TestClose_StopsHeartbeat(t *testing.T) {
	oldP := relayPingInterval
	relayPingInterval = 50 * time.Millisecond
	defer func() { relayPingInterval = oldP }()

	fr := newFakeRelay(t, 0, true, false)
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
	fr := newFakeRelay(t, 0, true, false)
	r := newTestRelayClient(t, fr)

	require.NoError(t, r.Connect())
	require.NoError(t, r.Connect())
	require.EqualValues(t, 1, atomic.LoadInt32(&fr.accepted))
	_ = r.Close()
}
