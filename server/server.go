package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
)

type Server struct {
	run      bool
	server   *http.Server
	logger   *zap.Logger
	cert     *tls.Certificate
	hostname string
	port     int
	handlers HandlerPatternMap
}

func NewServer(cert *tls.Certificate, hostname string) Server {
	return Server{logger: logutils.ZapLogger(), cert: cert, hostname: hostname}
}

func (s *Server) getHost() string {
	// TODO consider returning an error if s.getPort returns `0`, as this means that the listener is not ready
	return fmt.Sprintf("%s:%d", s.hostname, s.port)
}

func (s *Server) listenAndServe() {
	cfg := &tls.Config{Certificates: []tls.Certificate{*s.cert}, ServerName: s.hostname, MinVersion: tls.VersionTLS12}

	// in case of restart, we should use the same port as the first start in order not to break existing links
	listener, err := tls.Listen("tcp", s.getHost(), cfg)
	if err != nil {
		s.logger.Error("failed to start server, retrying", zap.Error(err))
		s.port = 0
		err = s.Start()
		if err != nil {
			s.logger.Error("server start failed, giving up", zap.Error(err))
		}
		return
	}

	s.port = listener.Addr().(*net.TCPAddr).Port
	s.run = true

	err = s.server.Serve(listener)
	if err != http.ErrServerClosed {
		s.logger.Error("server failed unexpectedly, restarting", zap.Error(err))
		err = s.Start()
		if err != nil {
			s.logger.Error("server start failed, giving up", zap.Error(err))
		}
		return
	}

	s.run = false
}

func (s *Server) resetServer() {
	s.server = new(http.Server)
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
	// Once Shutdown has been called on a server, it may not be reused;
	s.resetServer()
	s.applyHandlers()
	go s.listenAndServe()
	return nil
}

func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}

	return nil
}

func (s *Server) ToForeground() {
	if !s.run && (s.server != nil) {
		err := s.Start()
		if err != nil {
			s.logger.Error("server start failed during foreground transition", zap.Error(err))
		}
	}
}

func (s *Server) ToBackground() {
	if s.run {
		err := s.Stop()
		if err != nil {
			s.logger.Error("server stop failed during background transition", zap.Error(err))
		}
	}
}

func (s *Server) SetHandlers(handlers HandlerPatternMap) {
	s.handlers = handlers
}

func (s *Server) MakeBaseURL() *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   s.getHost(),
	}
}
