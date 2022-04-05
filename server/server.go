package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/logutils"
)

var globalCertificate *tls.Certificate = nil
var globalPem string

func generateTLSCert() error {
	if globalCertificate != nil {
		return nil
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	cert, err := GenerateX509Cert(notBefore, notAfter)
	if err != nil {
		return err
	}

	certPem, keyPem, err := GenerateX509PEMs(cert, priv)
	if err != nil {
		return err
	}

	finalCert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return err
	}

	globalCertificate = &finalCert
	globalPem = string(certPem)
	return nil
}

func PublicTLSCert() (string, error) {
	err := generateTLSCert()
	if err != nil {
		return "", err
	}

	return globalPem, nil
}

type Server struct {
	Port       int
	run        bool
	server     *http.Server
	logger     *zap.Logger
	db         *sql.DB
	cert       *tls.Certificate
	downloader *ipfs.Downloader
}

type Config struct {
	Cert *tls.Certificate
	Port int
}

func NewServer(db *sql.DB,  downloader *ipfs.Downloader, config *Config) (*Server, error) {
	s := &Server{db: db, logger: logutils.ZapLogger(), downloader: downloader}

	if config == nil {
		err := generateTLSCert()
		if err != nil {
			return nil, err
		}

		s.cert = globalCertificate
		s.Port = 0
	} else {
		s.cert = config.Cert
		s.Port = config.Port
	}

	return s, nil
}

func (s *Server) listenAndServe() {
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

func (s *Server) LoadHandlers(handlers HandlerPatternMap) {
	var hr *http.ServeMux
	if s.server != nil && s.server.Handler != nil {
		hr = s.server.Handler.(*http.ServeMux)
	} else {
		hr = http.NewServeMux()
	}

	for p, h := range handlers {
		hr.HandleFunc(p, h)
	}
	s.server = &http.Server{Handler: hr}
}

func (s *Server) LoadMediaHandlers() {
	s.LoadHandlers(HandlerPatternMap{
		"/messages/images": handleImage(s.db, s.logger),
		"/messages/audio": handleAudio(s.db, s.logger),
		"/messages/identicons": handleIdenticon(s.logger),
		"/ipfs": handleIPFS(s.downloader, s.logger),
	})
}
