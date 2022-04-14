package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
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

func GenerateX509Cert(sn *big.Int, from, to time.Time) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber:          sn,
		Subject:               pkix.Name{Organization: []string{"Self-signed cert"}},
		NotBefore:             from,
		NotAfter:              to,
		DNSNames:              []string{"localhost"},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
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

	cert := GenerateX509Cert(sn, notBefore, notAfter)
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

func GenerateCertFromKey(pk *ecdsa.PrivateKey, ttl time.Duration) (tls.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(ttl)

	cert := GenerateX509Cert(makeSerialNumberFromKey(pk), notBefore, notAfter)
	certPem, keyPem, err := GenerateX509PEMs(cert, pk)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(certPem, keyPem)
}
