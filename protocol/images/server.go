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
	"log"
	"math/big"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/identity/identicon"
)

var globalCertificate *tls.Certificate = nil
var globalPem string

func generateTLSCert() {
	if globalCertificate != nil {
		return
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Self-signed cert"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}

	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	finalCert, err := tls.X509KeyPair(certPem, keyPem)

	if err != nil {
		log.Fatalf("Unable to decode certificate: %v", err)
	}

	globalCertificate = &finalCert
	globalPem = string(certPem)
}

func PublicTLSCert() string {
	generateTLSCert()
	return globalPem
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

func NewServer(db *sql.DB, logger *zap.Logger) *Server {
	generateTLSCert()
	return &Server{db: db, logger: logger, cert: globalCertificate}
}

func (s *Server) Start() error {
	cfg := &tls.Config{Certificates: []tls.Certificate{*s.cert}, ServerName: "localhost"}
	listener, err := tls.Listen("tcp", "localhost:0", cfg)
	if err != nil {
		return err
	}
	s.Port = listener.Addr().(*net.TCPAddr).Port
	handler := http.NewServeMux()
	handler.Handle("/messages/images", &messageHandler{db: s.db, logger: s.logger})
	handler.Handle("/messages/identicons", &identiconHandler{logger: s.logger})
	s.server = &http.Server{Handler: handler}
	go func() {
		err := s.server.Serve(listener)
		if err != nil {
			s.logger.Error("failed to start server", zap.Error(err))
			return
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	return s.server.Shutdown(context.Background())
}
