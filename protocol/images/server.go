package images

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/identity/identicon"
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

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"Self-signed cert"}},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		DNSNames:              []string{"localhost"},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}

	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

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

type messageHandler struct {
	db     *sql.DB
	logger *zap.Logger
}

type identiconHandler struct {
	logger *zap.Logger
}

func (s *identiconHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pks, ok := r.URL.Query()["publicKey"]
	if !ok || len(pks) == 0 {
		s.logger.Error("no publicKey")
		return
	}
	pk := pks[0]
	image, err := identicon.Generate(pk)
	if err != nil {
		s.logger.Error("could not generate identicon")
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "max-age:290304000, public")
	w.Header().Set("Expires", time.Now().AddDate(60, 0, 0).Format(http.TimeFormat))

	_, err = w.Write(image)
	if err != nil {
		s.logger.Error("failed to write image", zap.Error(err))
	}
}

func (s *messageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	messageIDs, ok := r.URL.Query()["messageId"]
	if !ok || len(messageIDs) == 0 {
		s.logger.Error("no messageID")
		return
	}
	messageID := messageIDs[0]
	var image []byte
	err := s.db.QueryRow(`SELECT image_payload FROM user_messages WHERE id = ?`, messageID).Scan(&image)
	if err != nil {
		s.logger.Error("failed to find image", zap.Error(err))
		return
	}
	if len(image) == 0 {
		s.logger.Error("empty image")
		return
	}
	mime, err := ImageMime(image)
	if err != nil {
		s.logger.Error("failed to get mime", zap.Error(err))
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "no-store")

	_, err = w.Write(image)
	if err != nil {
		s.logger.Error("failed to write image", zap.Error(err))
	}
}

type Server struct {
	Port   int
	server *http.Server
	logger *zap.Logger
	db     *sql.DB
	cert   *tls.Certificate
}

func NewServer(db *sql.DB, logger *zap.Logger) (*Server, error) {
	err := generateTLSCert()

	if err != nil {
		return nil, err
	}

	return &Server{db: db, logger: logger, cert: globalCertificate, Port: 0}, nil
}

func (s *Server) listenAndServe() {
	cfg := &tls.Config{Certificates: []tls.Certificate{*s.cert}, ServerName: "localhost", MinVersion: tls.VersionTLS12}

	// in case of restart, we should use the same port as the first start in order not to break existing links
	addr := fmt.Sprintf("localhost:%d", s.Port)

	listener, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		s.logger.Error("failed to start server", zap.Error(err))
		return
	}

	s.Port = listener.Addr().(*net.TCPAddr).Port

	err = s.server.Serve(listener)
	if err != http.ErrServerClosed {
		s.logger.Error("server failed unexpectedly, restarting", zap.Error(err))
		go s.listenAndServe()
	}
}

func (s *Server) Start() error {
	handler := http.NewServeMux()
	handler.Handle("/messages/images", &messageHandler{db: s.db, logger: s.logger})
	handler.Handle("/messages/identicons", &identiconHandler{logger: s.logger})
	s.server = &http.Server{Handler: handler}

	go s.listenAndServe()

	return nil
}

func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}

	return nil
}
