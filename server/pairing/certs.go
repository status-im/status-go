package pairing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"time"

	"github.com/status-im/status-go/server"

	"github.com/status-im/status-go/signal"
)

var globalCertificate *tls.Certificate = nil
var globalPem string

func makeRandomSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}

func makeSerialNumberFromKey(pk *ecdsa.PrivateKey) *big.Int {
	h := sha256.New()
	h.Write(append(pk.D.Bytes(), append(pk.Y.Bytes(), pk.X.Bytes()...)...))

	return new(big.Int).SetBytes(h.Sum(nil))
}

func GenerateX509Cert(sn *big.Int, from, to time.Time, hostname string) *x509.Certificate {
	c := &x509.Certificate{
		SerialNumber:          sn,
		Subject:               pkix.Name{Organization: []string{"Self-signed cert"}},
		NotBefore:             from,
		NotAfter:              to,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	ip := net.ParseIP(hostname)
	if ip != nil {
		c.IPAddresses = []net.IP{ip}
	} else {
		c.DNSNames = []string{hostname}
	}

	return c
}

func GenerateX509PEMs(cert *x509.Certificate, key *ecdsa.PrivateKey) (certPem, keyPem []byte, err error) {
	derBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		return
	}
	certPem = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return
	}
	keyPem = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	return
}

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

	sn, err := makeRandomSerialNumber()
	if err != nil {
		return err
	}

	cert := GenerateX509Cert(sn, notBefore, notAfter, localhost)
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

func GenerateCertFromKey(pk *ecdsa.PrivateKey, from time.Time, hostname string) (tls.Certificate, []byte, error) {
	cert := GenerateX509Cert(makeSerialNumberFromKey(pk), from, from.Add(time.Hour), hostname)
	certPem, keyPem, err := GenerateX509PEMs(cert, pk)
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

// ToECDSA takes a []byte of D and uses it to create an ecdsa.PublicKey on the elliptic.P256 curve
// this function is basically a P256 curve version of eth-node/crypto.ToECDSA without all the nice validation
func ToECDSA(d []byte) *ecdsa.PrivateKey {
	k := new(ecdsa.PrivateKey)
	k.D = new(big.Int).SetBytes(d)
	k.PublicKey.Curve = elliptic.P256()

	k.PublicKey.X, k.PublicKey.Y = k.PublicKey.Curve.ScalarBaseMult(d)
	return k
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

	conn, err := tls.Dial("tcp", URL.Host, conf)
	if err != nil {
		signal.SendLocalPairingEvent(server.Event{Type: server.EventConnectionError, Error: err.Error(), Action: server.ActionConnect})
		return nil, err
	}
	defer conn.Close()

	// No error on the dial out then the URL.Host is accessible
	signal.SendLocalPairingEvent(server.Event{Type: server.EventConnectionSuccess, Action: server.ActionConnect})

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) != 1 {
		return nil, fmt.Errorf("expected 1 TLS certificate, received '%d'", len(certs))
	}

	return certs[0], nil
}
