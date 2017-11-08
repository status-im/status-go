package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// Server is started on demand by statusd to route JSON-RPC requests
// by the statusd-cli or other clients to the according components.
type Server struct {
	ctx           context.Context
	clientAddress string
	port          string
	server        *rpc.Server
	listener      net.Listener
}

// NewServer creates a new server by starting a listener routing
// the requests to their according handlers.
func NewServer(ctx context.Context, clientAddress, port string) (*Server, error) {
	s := &Server{
		ctx:           ctx,
		clientAddress: clientAddress,
		port:          port,
		server:        rpc.NewServer(),
	}
	s.server.HandleHTTP("/rpc", "/debug/rpc")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("router cannot create listener: %v", err)
	}
	s.listener = listener
	// Register the services.
	s.server.RegisterName("API", newAPIService())
	// Start listening to requests.
	go s.backend()
	return s, nil
}

// backend accepts the connections by the configured client
// and runs it to route the requests to the registered
// services.
func (s *Server) backend() {
	defer s.listener.Close()
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Printf("router cannot establish connection: %v", err)
				continue
			}
			if conn.RemoteAddr().String() != s.clientAddress {
				log.Printf("connection from invalid client '%s' rejected", conn.RemoteAddr().String())
				conn.Close()
				continue
			}
			go s.server.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}
}
