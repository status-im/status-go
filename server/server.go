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
	netIP    net.IP
	listener net.Listener
}

func NewServer(cert *tls.Certificate, ip net.IP) Server {
	return Server{logger: logutils.ZapLogger(), cert: cert, netIP: ip}
}

func (s *Server) setListener(l net.Listener) {
	s.listener = l
}

func (s *Server) resetListener() {
	s.listener = nil
}

// getPort depends on the Server.listener to provide a port number, net.Listener should determine the port.
// This is because there is no way to know what ports are available on the host device in advance
func (s *Server) getPort() int {
	if s.listener == nil {
		return 0
	}

	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *Server) listenAndServe() {
	cfg := &tls.Config{Certificates: []tls.Certificate{*s.cert}, ServerName: s.netIP.String(), MinVersion: tls.VersionTLS12}

	// in case of restart, we should use the same port as the first start in order not to break existing links
	addr := fmt.Sprintf("%s:%d", s.netIP, s.getPort())

	listener, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		s.logger.Error("failed to start server, retrying", zap.Error(err))
		s.resetListener()
		err = s.Start()
		if err != nil {
			s.logger.Error("server start failed, giving up", zap.Error(err))
		}
		return
	}

	s.setListener(listener)
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

func (s *Server) Start() error {
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

func (s *Server) WithHandlers(handlers HandlerPatternMap) {
	switch {
	case s.server != nil && s.server.Handler != nil:
		break
	case s.server != nil && s.server.Handler == nil:
		s.server.Handler = http.NewServeMux()
	default:
		s.server = &http.Server{}
		s.server.Handler = http.NewServeMux()
	}

	for p, h := range handlers {
		s.server.Handler.(*http.ServeMux).HandleFunc(p, h)
	}
}

func (s *Server) MakeBaseURL() *url.URL {
	// TODO consider returning an error if s.getPort returns `0`, as this means that the listener is not ready
	return &url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s:%d", s.netIP, s.getPort()),
	}
}
