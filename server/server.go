package server

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"net"
	"net/http"

	"go.uber.org/zap"
)

type Server struct {
	Port   int
	run    bool
	server *http.Server
	logger *zap.Logger
	db     *sql.DB
	cert   *tls.Certificate
}

type Option func(server *Server)

func SetCert(cert *tls.Certificate) func(*Server){
	return func(s *Server) {
		s.cert = cert
	}
}

func SetPort(port int) func(*Server){
	return func(s *Server){
		s.Port = port
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
		s.Port = 0
	} else {
		for _, cf := range configs {
			cf(s)
		}
	}

	return s, nil
}

func (s *Server) listenAndServe() {
	spew.Dump("listenAndServe")

	cfg := &tls.Config{Certificates: []tls.Certificate{*s.cert}, ServerName: "localhost", MinVersion: tls.VersionTLS12}

	// in case of restart, we should use the same port as the first start in order not to break existing links
	addr := fmt.Sprintf("localhost:%d", s.Port)

	listener, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		s.logger.Error("failed to start server, retrying", zap.Error(err))
		s.Port = 0
		err = s.Start()
		if err != nil {
			s.logger.Error("server start failed, giving up", zap.Error(err))
		}
		return
	}

	s.Port = listener.Addr().(*net.TCPAddr).Port
	s.run = true

	spew.Dump("pre : s.server.Serve(listener)", s)
	err = s.server.Serve(listener)
	spew.Dump("s.server.Serve(listener)", err)

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
		"/messages/images": handleImage(s.db, s.logger),
		"/messages/audio": handleAudio(s.db, s.logger),
		"/messages/identicons": handleIdenticon(s.logger),
	})
}
