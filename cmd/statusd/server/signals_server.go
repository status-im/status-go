package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ethereum/go-ethereum/log"

	statusgo "github.com/status-im/status-go/mobile"
	"github.com/status-im/status-go/signal"
	"io"
)

type Server struct {
	server      *http.Server
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

	http.HandleFunc("/signals", s.signals)
	s.addStatusGoEndpoint("/InitializeApplication", statusgo.InitializeApplication)
	s.addStatusGoEndpoint("/CreateAccountAndLogin", statusgo.CreateAccountAndLogin)
	s.addStatusGoEndpoint("/RestoreAccountAndLogin", statusgo.RestoreAccountAndLogin)
	s.addStatusGoEndpoint("/LoginAccount", statusgo.LoginAccount)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.address = listener.Addr().String()

	//go func() {
	err = s.server.Serve(listener)
	if !errors.Is(err, http.ErrServerClosed) {
		log.Error("signals server closed with error: %w", err)
	}
	//}()

	return nil
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

func (s *SignalsServer) addStatusGoEndpoint(endpoint string, handler func(string) string) {
	http.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
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
