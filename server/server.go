package server

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"sync"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
)

type Config struct {
	Cert     *tls.Certificate
	AddrPort netip.AddrPort

	// AdvertizeHost and AdvertizePort define the a different host/port to be advertized in media URLs than the one server listen on.
	// Can be used when status-go is running behind a NAT/PAT. If empty/zero, the actual listening address is used.
	AdvertizeHost string
	AdvertizePort int
}

type Server struct {
	lifecycleMu sync.Mutex

	listener net.Listener
	server   *http.Server
	logger   *zap.Logger
	handlers HandlerPatternMap

	config *Config

	// address is the host and port the server is listening on
	address *net.TCPAddr

	// isRunning is true if the server was started and is running
	isRunning bool

	// cachedPort stores the port from the first successful bind when AddrPort used port 0 (ephemeral).
	// Reused on pause/resume so URLs remain valid across ToBackground/ToForeground cycles.
	cachedPort int

	*timeoutManager
}

func NewServer(logger *zap.Logger, config *Config) Server {
	return Server{
		logger:         logger,
		config:         config,
		timeoutManager: newTimeoutManager(),
	}
}

func (s *Server) GetAddrPort() string {
	if s.address == nil {
		return ""
	}

	host := s.address.IP.String()
	if s.config != nil && s.config.Cert != nil && len(s.config.Cert.Leaf.DNSNames) > 0 {
		host = s.config.Cert.Leaf.DNSNames[0]
	}
	if s.config != nil && s.config.AdvertizeHost != "" {
		host = s.config.AdvertizeHost
	}

	return net.JoinHostPort(host, strconv.Itoa(s.GetPort()))
}

func (s *Server) GetPort() int {
	if s.address == nil {
		return 0
	}
	if s.config != nil && s.config.AdvertizePort != 0 {
		return s.config.AdvertizePort
	}
	return s.address.Port
}

func (s *Server) GetListeningAddrPort() string {
	if s.address == nil {
		return ""
	}
	return s.address.String()
}

func (s *Server) GetCert() *tls.Certificate {
	return s.config.Cert
}

func (s *Server) GetLogger() *zap.Logger {
	return s.logger
}

// getBindAddrPort returns the address to bind to. When config used port 0 (ephemeral)
// and we have a cached port from a previous run, reuse it so URLs stay stable across pause/resume.
func (s *Server) getBindAddrPort() netip.AddrPort {
	if s.cachedPort != 0 && s.config.AddrPort.Port() == 0 {
		return netip.AddrPortFrom(s.config.AddrPort.Addr(), uint16(s.cachedPort))
	}
	return s.config.AddrPort
}

func (s *Server) createListener() (net.Listener, error) {
	addr := s.getBindAddrPort()
	if s.config.Cert == nil {
		// HTTP mode
		return net.Listen("tcp", addr.String())
	}

	// HTTPS mode
	serverName := addr.Addr().String()
	if len(s.config.Cert.Leaf.DNSNames) > 0 {
		serverName = s.config.Cert.Leaf.DNSNames[0]
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{*s.config.Cert},
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}
	return tls.Listen("tcp", addr.String(), cfg)
}

func (s *Server) listen() error {
	s.address = nil

	var err error
	s.listener, err = s.createListener()
	if err != nil {
		s.logger.Error("failed to start server", zap.Error(err))
		return err
	}

	s.address = s.listener.Addr().(*net.TCPAddr)
	if s.config.AddrPort.Port() == 0 {
		s.cachedPort = s.address.Port
	}

	s.StartTimeout(func() {
		err := s.Stop()
		if err != nil {
			s.logger.Error("server termination fail", zap.Error(err))
		}
	})

	return nil
}

func (s *Server) serve() {
	defer common.LogOnPanic()

	defer func() {
		s.isRunning = false
		s.address = nil
	}()

	err := s.server.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return
	}

	s.logger.Error("server failed unexpectedly, restarting", zap.Error(err))
	return

}

func (s *Server) resetServer() {
	s.StopTimeout()
	s.server = new(http.Server)
	s.address = nil
}

func (s *Server) applyHandlers() {
	if s.server == nil {
		s.server = new(http.Server)
	}
	mux := http.NewServeMux()

	for p, h := range s.handlers {
		mux.HandleFunc(p, h)
	}
	s.server.Handler = mux
}

func (s *Server) Start() error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	if s.isRunning {
		return nil
	}

	// Once Shutdown has been called on a server, it may not be reused;
	s.resetServer()
	s.applyHandlers()

	err := s.listen()
	if err != nil {
		return err
	}

	// Mark running synchronously to avoid pause/play races where ToBackground
	// can run before serve() goroutine has a chance to set the state.
	s.isRunning = true
	go s.serve()
	return nil
}

func (s *Server) Stop() error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	s.StopTimeout()
	if !s.isRunning || s.server == nil {
		return nil
	}

	// Flip state before shutdown so rapid foreground/background transitions
	// don't attempt a concurrent second bind to a cached port.
	s.isRunning = false
	return s.server.Shutdown(context.Background())
}

func (s *Server) IsRunning() bool {
	return s.isRunning
}

func (s *Server) ToForeground() {
	err := s.Start()
	if err != nil {
		s.logger.Error("server start failed during foreground transition", zap.Error(err))
	}
}

func (s *Server) ToBackground() {
	err := s.Stop()
	if err != nil {
		s.logger.Error("server stop failed during background transition", zap.Error(err))
	}
}

func (s *Server) SetHandlers(handlers HandlerPatternMap) {
	s.handlers = handlers
}

func (s *Server) AddHandlers(handlers HandlerPatternMap) {
	if s.handlers == nil {
		s.handlers = make(HandlerPatternMap)
	}

	for name := range handlers {
		s.handlers[name] = handlers[name]
	}
}

func (s *Server) MakeBaseURL() *url.URL {
	if s.address == nil {
		return &url.URL{}
	}

	scheme := "http"
	if s.config != nil && s.config.Cert != nil {
		scheme = "https"
	}

	return &url.URL{
		Scheme: scheme,
		Host:   s.GetAddrPort(),
	}
}
