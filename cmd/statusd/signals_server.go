package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/signal"
)

type SignalsServer struct {
	server      *http.Server
	lock        sync.Mutex
	connections map[*websocket.Conn]struct{}
	address     string
}

func NewSignalsServer() *SignalsServer {
	return &SignalsServer{
		connections: make(map[*websocket.Conn]struct{}, 1),
	}
}

func (s *SignalsServer) Address() string {
	return s.address
}

func (s *SignalsServer) Setup() {
	signal.SetMobileSignalHandler(s.signalHandler)
}

func (s *SignalsServer) signalHandler(data []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for connection := range s.connections {
		err := connection.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Error("failed to write message: %w", err)
		}
	}
}

func (s *SignalsServer) Listen(address string) error {
	if s.server != nil {
		return errors.New("server already started")
	}

	s.server = &http.Server{
		Addr:              address,
		ReadHeaderTimeout: 5 * time.Second,
	}

	http.HandleFunc("/signals", s.signals)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.address = listener.Addr().String()

	go func() {
		err := s.server.Serve(listener)
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("signals server closed with error: %w", err)
		}
	}()

	return nil
}

func (s *SignalsServer) Stop(ctx context.Context) {
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

func (s *SignalsServer) signals(w http.ResponseWriter, r *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Accepting all requests
		},
	}

	connection, _ := upgrader.Upgrade(w, r, nil)
	s.connections[connection] = struct{}{}
}
