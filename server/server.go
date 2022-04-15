package server

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"

	"go.uber.org/zap"
)

var (
	defaultIp = net.IP{127, 0, 0, 1}
)

type Server struct {
	port   int
	run    bool
	server *http.Server
	logger *zap.Logger
	db     *sql.DB
	cert   *tls.Certificate
	netIp  net.IP
}

type Option func(server *Server)

func SetCert(cert *tls.Certificate) func(*Server) {
	return func(s *Server) {
		s.cert = cert
	}
}

func SetNetIP(ip net.IP) func(*Server) {
	return func(s *Server) {
		s.netIp = ip
	}
}

func SetPort(port int) func(*Server) {
	return func(s *Server) {
		s.port = port
	}
}

func NewServer(db *sql.DB, logger *zap.Logger, configs ...Option) (*Server, error) {
	s := &Server{db: db, logger: logger}

	if len(configs) == 0 {
		// default behaviour
		err := generateTLSCert()
		if err != nil {
			return nil, err
		}

		s.cert = globalCertificate
		s.netIp = defaultIp
		s.port = 0
	} else {
		for _, cf := range configs {
			cf(s)
		}
	}

	return s, nil
}

func (s *Server) listenAndServe() {
	cfg := &tls.Config{Certificates: []tls.Certificate{*s.cert}, ServerName: s.netIp.String(), MinVersion: tls.VersionTLS12}

	// in case of restart, we should use the same port as the first start in order not to break existing links
	addr := fmt.Sprintf("%s:%d", s.netIp, s.port)

	listener, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		s.logger.Error("failed to start server, retrying", zap.Error(err))
		s.port = 0 //TODO find out why the port is set to 0 here
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

func (s *Server) WithMediaHandlers() {
	s.WithHandlers(HandlerPatternMap{
		"/messages/images":     handleImage(s.db, s.logger),
		"/messages/audio":      handleAudio(s.db, s.logger),
		"/messages/identicons": handleIdenticon(s.logger),
	})
}

func (s *Server) MakeBaseURL() string {
	return fmt.Sprintf("https://%s:%d", s.netIp, s.port)
}

func (s *Server) MakeImageServerURL() string {
	return s.MakeBaseURL() + "/messages/"
}

func (s *Server) MakeIdenticonURL(from string) string {
	return s.MakeBaseURL() + "/messages/identicons?publicKey=" + from
}

func (s *Server) MakeImageURL(id string) string {
	return s.MakeBaseURL() + "/messages/images?messageId=" + id
}

func (s *Server) MakeAudioURL(id string) string {
	return s.MakeBaseURL() + "/messages/audio?messageId=" + id
}
