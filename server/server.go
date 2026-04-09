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
	"sync/atomic"
	"syscall"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
)

// tcpListenConfig sets SO_REUSEADDR so that after the previous listener is fully
// closed, the kernel can allow binding the same cached ephemeral port again (e.g.
// TIME_WAIT) without a long delay. It does not help if Stop returns before that
// socket is closed — Stop waits on serveWg for that ordering.
var tcpListenConfig = net.ListenConfig{
	Control: func(network, address string, c syscall.RawConn) error {
		var optErr error
		if err := c.Control(func(fd uintptr) {
			optErr = setReuseAddr(fd)
		}); err != nil {
			return err
		}
		return optErr
	},
}

type Config struct {
	Cert     *tls.Certificate
	AddrPort netip.AddrPort

	// AdvertizeHost and AdvertizePort define the a different host/port to be advertized in media URLs than the one server listen on.
	// Can be used when status-go is running behind a NAT/PAT. If empty/zero, the actual listening address is used.
	AdvertizeHost string
	AdvertizePort int
}

type Server struct {
	mu sync.Mutex

	listener net.Listener
	server   *http.Server
	logger   *zap.Logger
	handlers HandlerPatternMap

	config *Config

	// address is the host and port the server is listening on
	address *net.TCPAddr

	// running is true if the server was started and Serve is active; read via IsRunning().
	running atomic.Bool

	// cachedPort stores the port from the first successful bind when AddrPort used port 0 (ephemeral).
	// Reused on pause/resume so URLs remain valid across ToBackground/ToForeground cycles.
	cachedPort int

	// serveWg waits for the serve goroutine to finish all defers after http.Server.Serve returns,
	// so Stop does not return while the previous listener may still be tearing down (avoids EADDRINUSE
	// when rebinding the same cached ephemeral port).
	serveWg sync.WaitGroup

	*timeoutManager
}

func NewServer(logger *zap.Logger, config *Config) *Server {
	return &Server{
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
	addrStr := addr.String()

	ln, err := tcpListenConfig.Listen(context.Background(), "tcp", addrStr)
	if err != nil {
		return nil, err
	}

	if s.config.Cert == nil {
		return ln, nil
	}

	serverName := addr.Addr().String()
	if len(s.config.Cert.Leaf.DNSNames) > 0 {
		serverName = s.config.Cert.Leaf.DNSNames[0]
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{*s.config.Cert},
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}
	return tls.NewListener(ln, cfg), nil
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

func (s *Server) serve(currentServer *http.Server, currentListener net.Listener) {
	defer common.LogOnPanic()

	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		// If a newer Start() already replaced server/listener, do not clobber
		// the latest running instance state.
		if s.server == currentServer && s.listener == currentListener {
			s.running.Store(false)
			s.address = nil
		}
	}()

	err := currentServer.Serve(currentListener)
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
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
	s.running.Store(true)
	s.serveWg.Add(1)
	go func() {
		defer common.LogOnPanic()
		defer s.serveWg.Done()
		s.serve(s.server, s.listener)
	}()
	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	s.StopTimeout()
	if !s.running.Load() || s.server == nil {
		s.mu.Unlock()
		return nil
	}

	// Capture the current instance and release the lock before Shutdown.
	// Shutdown waits for Serve() to return, and Serve() may update state in
	// its defer path under the same mutex.
	currentServer := s.server
	s.running.Store(false)
	s.mu.Unlock()

	err := currentServer.Shutdown(context.Background())
	s.serveWg.Wait()
	return err
}

func (s *Server) IsRunning() bool {
	return s.running.Load()
}

func (s *Server) ToForeground() {
	if err := s.Start(); err != nil {
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
