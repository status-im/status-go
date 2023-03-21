package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

var globalCertificate *tls.Certificate = nil
var globalPem string

func makeRandomSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
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

	cert := GenerateX509Cert(sn, notBefore, notAfter, Localhost)
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

// ToECDSA takes a []byte of D and uses it to create an ecdsa.PublicKey on the elliptic.P256 curve
// this function is basically a P256 curve version of eth-node/crypto.ToECDSA without all the nice validation
func ToECDSA(d []byte) *ecdsa.PrivateKey {
	k := new(ecdsa.PrivateKey)
	k.D = new(big.Int).SetBytes(d)
	k.PublicKey.Curve = elliptic.P256()

	k.PublicKey.X, k.PublicKey.Y = k.PublicKey.Curve.ScalarBaseMult(d)
	return k
}
