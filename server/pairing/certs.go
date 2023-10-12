package pairing

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/server"
)

const CertificateMaxClockDrift = time.Minute

func makeSerialNumberFromKey(pk *ecdsa.PrivateKey) *big.Int {
	h := sha256.New()
	h.Write(append(pk.D.Bytes(), append(pk.Y.Bytes(), pk.X.Bytes()...)...))

	return new(big.Int).SetBytes(h.Sum(nil))
}

func GenerateCertFromKey(pk *ecdsa.PrivateKey, from time.Time, IPAddresses []net.IP, DNSNames []string) (tls.Certificate, []byte, error) {
	cert := server.GenerateX509Cert(makeSerialNumberFromKey(pk), from.Add(-CertificateMaxClockDrift), from.Add(time.Hour), IPAddresses, DNSNames)
	certPem, keyPem, err := server.GenerateX509PEMs(cert, pk)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	tlsCert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	block, _ := pem.Decode(certPem)
	if block == nil {
		return tls.Certificate{}, nil, fmt.Errorf("failed to decode certPem")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	tlsCert.Leaf = leaf

	return tlsCert, certPem, nil
}

// verifyCertPublicKey checks that the ecdsa.PublicKey using in a x509.Certificate matches a known ecdsa.PublicKey
func verifyCertPublicKey(cert *x509.Certificate, publicKey *ecdsa.PublicKey) error {
	certKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("unexpected public key type, expected ecdsa.PublicKey")
	}

	if !certKey.Equal(publicKey) {
		return fmt.Errorf("server certificate MUST match the given public key")
	}
	return nil
}

// verifyCertSig checks that a x509.Certificate's Signature verifies against x509.Certificate's PublicKey
// If the x509.Certificate's PublicKey is not an ecdsa.PublicKey an error will be thrown
func verifyCertSig(cert *x509.Certificate) (bool, error) {
	var esig struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(cert.Signature, &esig); err != nil {
		return false, err
	}

	hash := sha256.New()
	hash.Write(cert.RawTBSCertificate)

	ecKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("certificate public is not an ecdsa.PublicKey")
	}

	verified := ecdsa.Verify(ecKey, hash.Sum(nil), esig.R, esig.S)
	return verified, nil
}

// verifyCert verifies an x509.Certificate against a known ecdsa.PublicKey
// combining the checks of verifyCertPublicKey and verifyCertSig.
// If an x509.Certificate fails to verify an error is also thrown
func verifyCert(cert *x509.Certificate, publicKey *ecdsa.PublicKey) error {
	err := verifyCertPublicKey(cert, publicKey)
	if err != nil {
		return err
	}

	verified, err := verifyCertSig(cert)
	if err != nil {
		return err
	}
	if !verified {
		return fmt.Errorf("server certificate signature MUST verify")
	}
	return nil
}

// getServerCert pings a given tls host, extracts and returns its x509.Certificate
// the function expects there to be only 1 certificate
func getServerCert(URL *url.URL) (*x509.Certificate, error) {
	conf := &tls.Config{
		InsecureSkipVerify: true, // nolint: gosec // Only skip verify to get the server's TLS cert. DO NOT skip for any other reason.
	}

	// one second should be enough to get the server's TLS cert in LAN?
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: time.Second}, "tcp", URL.Host, conf)
	if err != nil {
		return nil, err
	}
	defer func(conn *tls.Conn) {
		if e := conn.Close(); e != nil {
			logutils.ZapLogger().Warn("failed to close temporary TLS connection:", zap.Error(e))
		}
	}(conn)

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) != 1 {
		return nil, fmt.Errorf("expected 1 TLS certificate, received '%d'", len(certs))
	}

	return certs[0], nil
}
