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
	listener net.Listener
	server   *http.Server
	logger   *zap.Logger
	handlers HandlerPatternMap

	config *Config

	// address is the host and port the server is listening on
	address *net.TCPAddr

	// isRunning is true if the server was started and is running
	isRunning bool

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

func (s *Server) createListener() (net.Listener, error) {
	if s.config.Cert == nil {
		// HTTP mode
		return net.Listen("tcp", s.config.AddrPort.String())
	}

	// HTTPS mode
	serverName := s.config.AddrPort.Addr().String()
	if len(s.config.Cert.Leaf.DNSNames) > 0 {
		serverName = s.config.Cert.Leaf.DNSNames[0]
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{*s.config.Cert},
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}
	return tls.Listen("tcp", s.config.AddrPort.String(), cfg)
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

	s.isRunning = true
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

	go s.serve()
	return nil
}

func (s *Server) Stop() error {
	s.StopTimeout()
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}

	return nil
}

func (s *Server) IsRunning() bool {
	return s.isRunning
}

func (s *Server) ToForeground() {
	if !s.isRunning && (s.server != nil) {
		err := s.Start()
		if err != nil {
			s.logger.Error("server start failed during foreground transition", zap.Error(err))
		}
	}
}

func (s *Server) ToBackground() {
	if s.isRunning {
		err := s.Stop()
		if err != nil {
			s.logger.Error("server stop failed during background transition", zap.Error(err))
		}
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
