package api

import (
	"errors"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"
	"time"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/log"
)

// Server is started on demand by statusd to route JSON-RPC requests
// by the statusd-cli or other clients to the according components.
type Server struct {
	validAddresses map[string]bool
	port           string
	rpcServer      *rpc.Server
	listener       net.Listener
	doneC          chan chan error
}

// ServeAPI creates and starts a new RPC server.
func ServeAPI(backend *api.StatusBackend, clientAddress, port string) (*Server, error) {
	s := &Server{
		port: port,
		validAddresses: map[string]bool{
			clientAddress: true,
			"localhost":   true,
			"127.0.0.1":   true,
			"[::1]":       true,
		},
		rpcServer: rpc.NewServer(),
		doneC:     make(chan chan error),
	}
	// Prepare RPC server.
	s.rpcServer.RegisterName("Admin", newAdminService())          //nolint: errcheck
	s.rpcServer.RegisterName("Status", newStatusService(backend)) //nolint: errcheck
	s.rpcServer.HandleHTTP("/rpc", "/debug/rpc")
	// Prepare listener and start backend.
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}
	s.listener = l
	go s.backend()
	return s, nil
}

// Close teminates the server.
func (s *Server) Close() error {
	errc := make(chan error)
	select {
	case s.doneC <- errc:
	default:
		// Channel is closed.
		return errors.New("API server already closed")
	}
	select {
	case err := <-errc:
		return err
	case <-time.After(30 * time.Second):
		return errors.New("timeout during server closing")
	}
}

// backend accepts the connections by the configured client
// and runs it to route the requests to the registered
// services.
func (s *Server) backend() {
	for {
		select {
		case errc := <-s.doneC:
			err := s.listener.Close()
			errc <- err
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Warn("API server cannot establish connection", "err", err)
				continue
			}
			remoteAddr := conn.RemoteAddr().String()
			remoteAddr = remoteAddr[:strings.LastIndex(remoteAddr, ":")]
			if !s.validAddresses[remoteAddr] {
				log.Error("connection from invalid client rejected", "addr", remoteAddr)
				conn.Close() //nolint: errcheck
				continue
			}
			go s.rpcServer.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}
}
