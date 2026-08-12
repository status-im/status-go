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
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
)

// backgroundShutdownTimeout bounds how long Server.ToBackground waits for
// active HTTP handlers to drain before force-closing remaining connections.
const backgroundShutdownTimeout = 300 * time.Millisecond

// teardownShutdownTimeout bounds the same drain at teardown. It is generous
// enough for normal in-flight requests to finish, but a handler that never
// returns must not be able to wedge Stop — and with it the logout path that
// holds the backend mutex.
const teardownShutdownTimeout = 3 * time.Second

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

	// serveDone is closed when the current serve goroutine has fully exited.
	// The dead-listener rebind path waits on the instance it captured — never
	// on serveWg, which a concurrent Start could grow while the mutex is
	// released, making the wait block on an unrelated healthy server.
	serveDone chan struct{}

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
	host := ""
	if s.address != nil {
		host = s.address.IP.String()
	} else if s.config != nil {
		host = s.config.AddrPort.Addr().String()
	}

	if s.config != nil && s.config.Cert != nil && len(s.config.Cert.Leaf.DNSNames) > 0 {
		host = s.config.Cert.Leaf.DNSNames[0]
	}
	if s.config != nil && s.config.AdvertizeHost != "" {
		host = s.config.AdvertizeHost
	}

	port := s.GetPort()
	if host == "" || port == 0 {
		return ""
	}

	return net.JoinHostPort(host, strconv.Itoa(port))
}

func (s *Server) GetPort() int {
	if s.config != nil && s.config.AdvertizePort != 0 {
		return s.config.AdvertizePort
	}

	if s.address != nil {
		return s.address.Port
	}

	if s.cachedPort != 0 {
		return s.cachedPort
	}

	if s.config != nil && s.config.AddrPort.Port() != 0 {
		return int(s.config.AddrPort.Port())
	}

	return 0
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

// listenerAlive verifies the bound listener actually accepts connections.
// On iOS a process suspension can kill the listening socket without the accept
// loop ever returning an error, leaving `running` true while every client
// connect is refused (observed when the app is suspended on the login screen,
// before the pausable-services bridge has anything to drive).
//
// A raw TCP dial is sufficient and safe with a TLS listener: crypto/tls only
// handshakes on the first read/write of an accepted conn, and net/http runs
// that in a per-connection goroutine — an immediate close surfaces (at most) a
// per-conn handshake error, never an Accept error, so Serve keeps running.
func (s *Server) listenerAlive() bool {
	addr := s.GetListeningAddrPort()
	if addr == "" {
		return false
	}
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		if s.listenerAlive() {
			return nil
		}
		s.logger.Warn("server marked running but listener is dead; rebinding")
		currentServer := s.server
		done := s.serveDone
		s.running.Store(false)
		_ = currentServer.Close()
		// The serve goroutine's deferred cleanup takes s.mu — release it while
		// waiting, and wait only on the captured instance, never on serveWg
		// (a concurrent Start could grow it with a healthy new server).
		s.mu.Unlock()
		if done != nil {
			<-done
		}
		s.mu.Lock()
		// Re-validate after the unlocked window: a concurrent Start may have
		// already rebound — in that case this call's work is done.
		if s.running.Load() {
			return nil
		}
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
	done := make(chan struct{})
	s.serveDone = done
	s.serveWg.Add(1)
	go func() {
		defer common.LogOnPanic()
		defer close(done)
		defer s.serveWg.Done()
		s.serve(s.server, s.listener)
	}()
	return nil
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), teardownShutdownTimeout)
	defer cancel()

	err := s.stopWith(ctx)
	if shouldForceCloseOnShutdownError(err) {
		s.logger.Warn("server graceful shutdown did not complete in time; forced close",
			zap.Error(err), zap.Duration("timeout", teardownShutdownTimeout))
		return nil
	}
	return err
}

// stopWith performs the graceful shutdown sequence bounded by ctx. If ctx
// expires or is canceled before active connections drain, the server is
// force-closed via Close so the caller isn't blocked by a slow or stuck
// handler. Callers always pass a deadline: a generous one at teardown, a short
// one for lifecycle events like pause/ToBackground.
func (s *Server) stopWith(ctx context.Context) error {
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

	err := currentServer.Shutdown(ctx)
	if shouldForceCloseOnShutdownError(err) {
		_ = currentServer.Close()
	}
	s.serveWg.Wait()
	return err
}

func shouldForceCloseOnShutdownError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
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
	ctx, cancel := context.WithTimeout(context.Background(), backgroundShutdownTimeout)
	defer cancel()
	if err := s.stopWith(ctx); err != nil {
		if shouldForceCloseOnShutdownError(err) {
			s.logger.Warn("server graceful shutdown did not complete in time; forced close",
				zap.Error(err), zap.Duration("timeout", backgroundShutdownTimeout))
			return
		}

		s.logger.Error("server shutdown failed during background transition", zap.Error(err))
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
	hostPort := s.GetAddrPort()
	if hostPort == "" {
		return &url.URL{}
	}

	scheme := "http"
	if s.config != nil && s.config.Cert != nil {
		scheme = "https"
	}

	return &url.URL{
		Scheme: scheme,
		Host:   hostPort,
	}
}
