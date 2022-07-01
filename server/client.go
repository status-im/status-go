package server

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type PairingClient struct {
	*http.Client

	baseAddress    *url.URL
	certPEM        []byte
	privateKey     *ecdsa.PrivateKey
	serverMode     Mode
	PayloadManager PayloadManager
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
