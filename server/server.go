package server

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"go.uber.org/zap"

	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/logutils"
)

const (
	basePath       = "/messages"
	identiconsPath = basePath + "/identicons"
	imagesPath     = basePath + "/images"
	audioPath      = basePath + "/audio"
	ipfsPath       = "/ipfs"
)

var (
	defaultIp = net.IP{127, 0, 0, 1}
)

type Server struct {
	port       int
	run        bool
	server     *http.Server
	logger     *zap.Logger
	db         *sql.DB
	cert       *tls.Certificate
	netIp      net.IP
	downloader *ipfs.Downloader
}

type Config struct {
	Cert *tls.Certificate
	NetIp net.IP
	Port int
}

// NewServer returns a *Server. If the config param is nil the default Server values are applied to the new Server
// otherwise the config params are applied to the Server.
func NewServer(db *sql.DB, downloader *ipfs.Downloader, config *Config) (*Server, error) {
	s := &Server{db: db, logger: logutils.ZapLogger(), downloader: downloader}

	if config == nil {
		err := generateTLSCert()
		if err != nil {
			return nil, err
		}

		s.cert = globalCertificate
		s.netIp = defaultIp
		s.port = 0
	} else {
		s.cert = config.Cert
		s.netIp = config.NetIp
		s.port = config.Port
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
		imagesPath:     handleImage(s.db, s.logger),
		audioPath:      handleAudio(s.db, s.logger),
		identiconsPath: handleIdenticon(s.logger),
		ipfsPath:       handleIPFS(s.downloader, s.logger),
	})
}

func (s *Server) MakeBaseURL() *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s:%d", s.netIp, s.port),
	}
}

func (s *Server) MakeImageServerURL() string {
	u := s.MakeBaseURL()
	u.Path = basePath + "/"
	return u.String()
}

func (s *Server) MakeIdenticonURL(from string) string {
	u := s.MakeBaseURL()
	u.Path = identiconsPath
	u.RawQuery = url.Values{"publicKey": {from}}.Encode()

	return u.String()
}

func (s *Server) MakeImageURL(id string) string {
	u := s.MakeBaseURL()
	u.Path = imagesPath
	u.RawQuery = url.Values{"messageId": {id}}.Encode()

	return u.String()
}

func (s *Server) MakeAudioURL(id string) string {
	u := s.MakeBaseURL()
	u.Path = audioPath
	u.RawQuery = url.Values{"messageId": {id}}.Encode()

	return u.String()
}
