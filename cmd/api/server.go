package api

import (
	"context"
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"
	"sync"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/log"
)

// Server is started on demand by statusd to route JSON-RPC requests
// by the statusd-cli or other clients to the according components.
type Server struct {
	ctx            context.Context
	mu             sync.Mutex
	validAddresses map[string]bool
	port           string
	server         *rpc.Server
	listener       net.Listener
	err            error
}

// NewServer creates a new server by starting a listener routing
// the requests to their according handlers.
func NewServer(ctx context.Context, backend *api.StatusBackend, clientAddress, port string) (*Server, error) {
	s := &Server{
		ctx:    ctx,
		port:   port,
		server: rpc.NewServer(),
	}
	s.validAddresses = map[string]bool{
		clientAddress: true,
		"localhost":   true,
		"127.0.0.1":   true,
		"[::1]":       true,
	}
	s.server.HandleHTTP("/rpc", "/debug/rpc")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("router cannot create listener: %v", err)
	}
	s.listener = listener
	// Register the services.
	s.server.RegisterName("Admin", newAdminService())
	s.server.RegisterName("Status", newStatusService(backend))
	// Start listening to requests.
	go s.backend()
	return s, nil
}

// Err returns nil if the server is running and everything is fine.
// In case it's stopped it returns the reason which is one of the
// context reasons.
func (s *Server) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// backend accepts the connections by the configured client
// and runs it to route the requests to the registered
// services.
func (s *Server) backend() {
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		lerr := s.listener.Close()
		s.listener = nil
		if s.err == nil {
			// Set possible error, not overwrite a given one.
			s.err = lerr
		}
	}()
	for {
		select {
		case <-s.ctx.Done():
			s.err = s.ctx.Err()
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Warn("router cannot establish connection", "err", err)
				continue
			}
			remoteAddr := conn.RemoteAddr().String()
			remoteAddr = remoteAddr[:strings.LastIndex(remoteAddr, ":")]
			log.Info("establishing client connection", "addr", remoteAddr)
			if !s.validAddresses[remoteAddr] {
				log.Error("connection from invalid client rejected", "addr", remoteAddr)
				conn.Close()
				continue
			}
			go s.server.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}
}
