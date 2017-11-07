package api

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// Router is started on demand by statusd to route JSON-RPC requests
// by the statusd-cli or other clients to the according components.
type Router struct {
	clientAddress string
	port          string
	server        *rpc.Server
	listener      net.Listener
	doneC         chan struct{}
}

// NewRouter creates a new router by starting a listener routing the
// requests to their according handlers.
func NewRouter(clientAddress, port string) (*Router, error) {
	r := &Router{
		clientAddress: clientAddress,
		port:          port,
		server:        rpc.NewServer(),
		doneC:         make(chan struct{}),
	}
	r.server.HandleHTTP("/rpc", "/debug/rpc")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("router cannot create listener: %v", err)
	}
	r.listener = listener
	// Register the services.
	r.server.Register(NewAPI())
	// Start listening to requests.
	go r.backend()
	return r, nil
}

// Stop closes the listener so that no RPCs are routed anymore.
func (r *Router) Stop() {
	close(r.doneC)
}

// backend accepts the connections by the configured client
// and runs it to route the requests to the registered
// services.
func (r *Router) backend() {
	defer r.listener.Close()
	for {
		select {
		case <-r.doneC:
			return
		default:
			conn, err := r.listener.Accept()
			if err != nil {
				log.Printf("router cannot establish connection: %v", err)
				continue
			}
			if conn.RemoteAddr().String() != r.clientAddress {
				log.Printf("connection from invalid client '%s' rejected", conn.RemoteAddr().String())
				conn.Close()
				continue
			}
			go r.server.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}
}
