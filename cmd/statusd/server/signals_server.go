package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/signal"
)

type Server struct {
	server      *http.Server
	listener    net.Listener
	mux         *http.ServeMux
	lock        sync.Mutex
	connections map[*websocket.Conn]struct{}
	address     string
}

func NewServer() *Server {
	return &Server{
		connections: make(map[*websocket.Conn]struct{}, 1),
	}
}

func (s *Server) Address() string {
	return s.address
}

func (s *Server) Port() (int, error) {
	_, portString, err := net.SplitHostPort(s.address)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(portString)
}

func (s *Server) Setup() {
	signal.SetMobileSignalHandler(s.signalHandler)
}

func (s *Server) signalHandler(data []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for connection := range s.connections {
		err := connection.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Error("failed to write message: %w", err)
		}
	}
}

func (s *Server) Listen(address string) error {
	if s.server != nil {
		return errors.New("server already started")
	}

	s.server = &http.Server{
		Addr:              address,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/signals", s.signals)
	s.server.Handler = s.mux

	var err error
	s.listener, err = net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.address = s.listener.Addr().String()

	return nil
}

func (s *Server) Serve() {
	err := s.server.Serve(s.listener)
	if !errors.Is(err, http.ErrServerClosed) {
		log.Error("signals server closed with error: %w", err)
	}
}

func (s *Server) Stop(ctx context.Context) {
	for connection := range s.connections {
		err := connection.Close()
		if err != nil {
			log.Error("failed to close connection: %w", err)
		}
		delete(s.connections, connection)
	}

	err := s.server.Shutdown(ctx)
	if err != nil {
		log.Error("failed to shutdown signals server: %w", err)
	}

	s.server = nil
	s.address = ""
}

func (s *Server) signals(w http.ResponseWriter, r *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Accepting all requests
		},
	}

	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("failed to upgrade connection: %w", err)
		return
	}

	s.connections[connection] = struct{}{}
}

func (s *Server) addEndpointWithResponse(name string, handler func(string) string) {
	log.Info("adding endpoint", "name", name)
	s.mux.HandleFunc(name, func(w http.ResponseWriter, r *http.Request) {
		request, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("failed to read request: %w", err)
			return
		}

		response := handler(string(request))

		_, err = w.Write([]byte(response))
		if err != nil {
			log.Error("failed to write response: %w", err)
		}
	})
}

func (s *Server) addEndpointNoRequest(name string, handler func() string) {
	log.Debug("adding endpoint", "name", name)
	s.mux.HandleFunc(name, func(w http.ResponseWriter, r *http.Request) {
		response := handler()

		_, err := w.Write([]byte(response))
		if err != nil {
			log.Error("failed to write response: %w", err)
		}
	})
}

func (s *Server) addUnsupportedEndpoint(name string) {
	log.Debug("marking unsupported endpoint", "name", name)
	s.mux.HandleFunc(name, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	})
}

func (s *Server) RegisterMobileAPI() {
	for name, endpoint := range EndpointsWithResponse {
		s.addEndpointWithResponse(name, endpoint)
	}
	for name, endpoint := range EndpointsNoRequest {
		s.addEndpointNoRequest(name, endpoint)
	}
	for _, name := range EndpointsUnsupported {
		s.addUnsupportedEndpoint(name)
	}
}
