package server

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
)

type PairingClient struct {
	*http.Client
	PayloadManager

	baseAddress *url.URL
	certPEM     []byte
	privateKey  *ecdsa.PrivateKey
	serverMode  Mode
	serverCert  *x509.Certificate
}

func NewPairingClient(c *ConnectionParams, config *PairingPayloadManagerConfig) (*PairingClient, error) {
	u, certPem, err := c.Generate()
	if err != nil {
		return nil, err
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	if ok := rootCAs.AppendCertsFromPEM(certPem); !ok {
		return nil, fmt.Errorf("failed to append certPem to rootCAs")
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false, // MUST BE FALSE
			RootCAs:            rootCAs,
		},
	}

	pm, err := NewPairingPayloadManager(c.privateKey, config)
	if err != nil {
		return nil, err
	}

	return &PairingClient{
		Client:         &http.Client{Transport: tr},
		baseAddress:    u,
		certPEM:        certPem,
		privateKey:     c.privateKey,
		serverMode:     c.serverMode,
		PayloadManager: pm,
	}, nil
}

func (c *PairingClient) PairAccount() error {
	switch c.serverMode {
	case Receiving:
		return c.sendAccountData()
	case Sending:
		return c.receiveAccountData()
	default:
		return fmt.Errorf("unrecognised server mode '%d'", c.serverMode)
	}
}

func (c *PairingClient) sendAccountData() error {
	c.baseAddress.Path = pairingReceive
	_, err := c.Post(c.baseAddress.String(), "application/octet-stream", bytes.NewBuffer(c.PayloadManager.ToSend()))
	if err != nil {
		return err
	}

	return nil
}

func (c *PairingClient) receiveAccountData() error {
	c.baseAddress.Path = pairingSend
	resp, err := c.Get(c.baseAddress.String())
	if err != nil {
		return err
	}

	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return c.PayloadManager.Receive(payload)
}

func verifyCertSig(cert *x509.Certificate) (bool, error) {
	var esig struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(cert.Signature, &esig); err != nil {
		return false, err
	}

	hash := sha256.New()
	hash.Write(cert.RawTBSCertificate)

	verified := ecdsa.Verify(cert.PublicKey.(*ecdsa.PublicKey), hash.Sum(nil), esig.R, esig.S)
	return verified, nil
}

func (c *PairingClient) getServerCert() error {
	conf := &tls.Config{
		InsecureSkipVerify: true, // Only skip verify to get the server's TLS cert. DO NOT skip for any other reason.
	}

	conn, err := tls.Dial("tcp", c.baseAddress.Host, conf)
	if err != nil {
		return err
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) != 1 {
		return fmt.Errorf("expected 1 TLS certificate, received '%d'", len(certs))
	}

	certKey, ok := certs[0].PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("unexpected public key type, expected ecdsa.PublicKey")
	}

	if certKey.Equal(c.privateKey) {
		return fmt.Errorf("server certificate MUST match the given public key")
	}

	verified, err := verifyCertSig(certs[0])
	if err != nil {
		return err
	}
	if !verified {
		return fmt.Errorf("server certificate signature MUST verify")
	}

	c.serverCert = certs[0]
	return nil
}
