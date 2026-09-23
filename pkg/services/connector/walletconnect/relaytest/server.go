// Package relaytest provides an in-process WalletConnect relay whose network
// behaviour can be switched at runtime, for tests only.
package relaytest

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Mode is how the relay treats new connections.
type Mode int32

const (
	// Healthy completes handshakes and answers every request.
	Healthy Mode = iota
	// Blackhole accepts TCP connections and never answers, like a lossy network.
	Blackhole
	// Reject answers handshakes with HTTP 503.
	Reject
)

// Server is a switchable relay listening on 127.0.0.1.
type Server struct {
	mode       atomic.Int32
	subscribes atomic.Int32
	held       atomic.Int32

	ln  net.Listener
	srv *http.Server

	mu    sync.Mutex
	conns []net.Conn
	ws    map[*websocket.Conn]struct{}
}

// New starts a relay in the given mode and stops it when the test ends.
func New(t testing.TB, mode Mode) *Server {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("relaytest: listen: %v", err)
	}
	s := &Server{ln: ln, ws: make(map[*websocket.Conn]struct{})}
	s.mode.Store(int32(mode))
	s.srv = &http.Server{Handler: http.HandlerFunc(s.serve), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.srv.Serve(gateListener{Listener: ln, s: s}) }()
	t.Cleanup(s.close)
	return s
}

// URL is the relay's WebSocket URL.
func (s *Server) URL() string { return "ws://" + s.ln.Addr().String() }

// SetMode changes how new connections are treated; open connections are kept.
func (s *Server) SetMode(m Mode) { s.mode.Store(int32(m)) }

// DropAll closes every open WebSocket connection.
func (s *Server) DropAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.ws {
		_ = c.Close()
		delete(s.ws, c)
	}
}

// Subscribes counts answered irn_subscribe requests.
func (s *Server) Subscribes() int32 { return s.subscribes.Load() }

// Held counts connections swallowed in Blackhole mode.
func (s *Server) Held() int32 { return s.held.Load() }

func (s *Server) close() {
	_ = s.srv.Close()
	s.DropAll()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.conns {
		_ = c.Close()
	}
	s.conns = nil
}

type gateListener struct {
	net.Listener
	s *Server
}

func (l gateListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if Mode(l.s.mode.Load()) != Blackhole {
			return c, nil
		}
		l.s.mu.Lock()
		l.s.conns = append(l.s.conns, c)
		l.s.mu.Unlock()
		l.s.held.Add(1)
	}
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	if Mode(s.mode.Load()) == Reject {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.ws[conn] = struct{}{}
	s.mu.Unlock()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(msg, &req) != nil || len(req.ID) == 0 {
			continue
		}
		result := "true"
		if req.Method == "irn_subscribe" {
			s.subscribes.Add(1)
			result = `"sub-id"`
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, result)
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if conn.WriteMessage(websocket.TextMessage, []byte(resp)) != nil {
			return
		}
	}
}
