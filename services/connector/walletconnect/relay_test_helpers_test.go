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

// fakeRelayOpts configures the in-process WalletConnect IRN-like WebSocket server.
type fakeRelayOpts struct {
	forceDropAt   int32 // >0: close the N-th accepted connection immediately after upgrade
	echoSubscribe bool  // reply to irn_subscribe with a fake "sub-id"
	silentOnPing  bool  // do not respond to ping (used by heartbeat tests)
}

// fakeRelay is an in-process WalletConnect IRN-like WebSocket server for relay tests.
type fakeRelay struct {
	Server *httptest.Server
	opts   fakeRelayOpts

	mu          sync.Mutex
	currentConn *websocket.Conn

	accepted int32
}

func newFakeRelay(t *testing.T, opts fakeRelayOpts) *fakeRelay {
	t.Helper()
	fr := &fakeRelay{opts: opts}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		n := atomic.AddInt32(&fr.accepted, 1)
		if fr.opts.forceDropAt > 0 && n == fr.opts.forceDropAt {
			_ = conn.Close()
			return
		}
		if fr.opts.silentOnPing {
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
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if !fr.opts.echoSubscribe {
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
