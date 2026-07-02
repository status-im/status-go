package server

import (
	"context"
	"expvar"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/pkg/errors"

	"github.com/status-im/status-go/cmd/status-backend/server/api"
	"github.com/status-im/status-go/signal"
)

func init() {
	expvar.Publish("numThreads", expvar.Func(func() any {
		threadsCount, _ := runtime.ThreadCreateProfile([]runtime.StackRecord{})
		return threadsCount
	}))
	// numGoroutine may already be published by github.com/anacrolix/envpprof
	// when the torrent history-archive backend is compiled in (use_torrent).
	// Publish it ourselves only when that dependency isn't linked, so the
	// metric is always available at /debug/vars regardless of build tags.
	if expvar.Get("numGoroutine") == nil {
		expvar.Publish("numGoroutine", expvar.Func(func() any {
			return runtime.NumGoroutine()
		}))
	}
}

type Server struct {
	server      *http.Server
	listener    net.Listener
	mux         *http.ServeMux
	lock        sync.Mutex
	connections map[*websocket.Conn]struct{}
	address     string
	config      *Config
	logger      *zap.Logger
}

func NewServer(logger *zap.Logger, options ...Option) *Server {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Server{
		connections: make(map[*websocket.Conn]struct{}, 1),
		config:      config,
		logger:      logger,
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

	deleteConnection := func(connection *websocket.Conn) {
		delete(s.connections, connection)
		err := connection.Close()
		if err != nil {
			s.logger.Error("failed to close connection", zap.Error(err))
		}
	}

	for connection := range s.connections {
		err := connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			s.logger.Error("failed to set write deadline", zap.Error(err))
			deleteConnection(connection)
			continue
		}

		err = connection.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			s.logger.Error("failed to write signal message", zap.Error(err))
			deleteConnection(connection)
		}
	}
}

func (s *Server) Listen(address string) error {
	if s.server != nil {
		return errors.New("server already started")
	}

	_, _, err := net.SplitHostPort(address)
	if err != nil {
		return errors.Wrap(err, "invalid address")
	}

	s.server = &http.Server{
		Addr:              address,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/health", api.Health)
	s.mux.HandleFunc("/signals", s.signals)
	s.server.Handler = s.mux

	// Register pprof handlers
	if s.config.profilingEnabled {
		runtime.SetMutexProfileFraction(1)
		s.mux.HandleFunc("/debug/pprof/", pprof.Index)
		s.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		s.mux.HandleFunc("/debug/vars", http.DefaultServeMux.ServeHTTP)
	}

	s.mux.HandleFunc("/statusgo/debug/FreeOSMemory", s.freeOSMemory)

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
		s.logger.Error("signals server closed with error: %w", zap.Error(err))
	}
}

func (s *Server) Stop(ctx context.Context) {
	for connection := range s.connections {
		err := connection.Close()
		if err != nil {
			s.logger.Error("failed to close connection: %w", zap.Error(err))
		}
		delete(s.connections, connection)
	}

	err := s.server.Shutdown(ctx)
	if err != nil {
		s.logger.Error("failed to shutdown signals server: %w", zap.Error(err))
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
		s.logger.Error("failed to upgrade connection: %w", zap.Error(err))
		return
	}
	s.logger.Debug("new websocket connection")

	s.connections[connection] = struct{}{}
}

func (s *Server) addEndpointWithResponse(name string, handler func(string) string) {
	s.logger.Debug("adding endpoint", zap.String("name", name))
	s.mux.HandleFunc(name, func(w http.ResponseWriter, r *http.Request) {
		request, err := io.ReadAll(r.Body)
		if err != nil {
			s.logger.Error("failed to read request: %w", zap.Error(err))
			return
		}

		response := handler(string(request))

		s.setHeaders(name, w)

		_, err = w.Write([]byte(response))
		if err != nil {
			s.logger.Error("failed to write response: %w", zap.Error(err))
		}
	})
}

func (s *Server) addEndpointNoRequest(name string, handler func() string) {
	s.logger.Debug("adding endpoint", zap.String("name", name))
	s.mux.HandleFunc(name, func(w http.ResponseWriter, r *http.Request) {
		response := handler()

		s.setHeaders(name, w)

		_, err := w.Write([]byte(response))
		if err != nil {
			s.logger.Error("failed to write response: %w", zap.Error(err))
		}
	})
}

func (s *Server) addUnsupportedEndpoint(name string) {
	s.logger.Debug("marking unsupported endpoint", zap.String("name", name))
	s.mux.HandleFunc(name, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	})
}

func (s *Server) RegisterMobileAPI() {
	for name, endpoint := range EndpointsWithRequest {
		s.addEndpointWithResponse(name, endpoint)
	}
	for name, endpoint := range EndpointsWithoutRequest {
		s.addEndpointNoRequest(name, endpoint)
	}
	for _, name := range EndpointsUnsupported {
		s.addUnsupportedEndpoint(name)
	}
}

func (s *Server) setHeaders(name string, w http.ResponseWriter) {
	if _, ok := EndpointsDeprecated[name]; ok {
		w.Header().Set("Deprecation", "true")
	}
	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) freeOSMemory(w http.ResponseWriter, r *http.Request) {
	debug.FreeOSMemory()
	w.WriteHeader(http.StatusOK)
}
