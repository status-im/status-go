package server

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/btcsuite/btcutil/base58"

	"github.com/status-im/status-go/multiaccounts"
)

type PairingClient struct {
	*http.Client
	PayloadManager

	baseAddress     *url.URL
	certPEM         []byte
	serverPK        *ecdsa.PublicKey
	serverMode      Mode
	serverCert      *x509.Certificate
	serverChallenge []byte
}

func NewPairingClient(c *ConnectionParams, config *PairingPayloadManagerConfig) (*PairingClient, error) {
	u, err := c.URL()
	if err != nil {
		return nil, err
	}

	serverCert, err := getServerCert(u)
	if err != nil {
		return nil, err
	}
	err = verifyCert(serverCert, c.publicKey)
	if err != nil {
		return nil, err
	}
	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCert.Raw})

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

	cj, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	pm, err := NewPairingPayloadManager(c.aesKey, config)
	if err != nil {
		return nil, err
	}

	return &PairingClient{
		Client:         &http.Client{Transport: tr, Jar: cj},
		baseAddress:    u,
		certPEM:        certPem,
		serverCert:     serverCert,
		serverPK:       c.publicKey,
		serverMode:     c.serverMode,
		PayloadManager: pm,
	}, nil
}

func (c *PairingClient) PairAccount() error {
	switch c.serverMode {
	case Receiving:
		return c.sendAccountData()
	case Sending:
		err := c.getChallenge()
		if err != nil {
			return err
		}
		return c.receiveAccountData()
	default:
		return fmt.Errorf("unrecognised server mode '%d'", c.serverMode)
	}
}

func (c *PairingClient) sendAccountData() error {
	err := c.Mount()
	if err != nil {
		return err
	}

	c.baseAddress.Path = pairingReceive
	resp, err := c.Post(c.baseAddress.String(), "application/octet-stream", bytes.NewBuffer(c.PayloadManager.ToSend()))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status not ok, received '%s'", resp.Status)
	}

	return nil
}

func (c *PairingClient) receiveAccountData() error {
	c.baseAddress.Path = pairingSend
	req, err := http.NewRequest(http.MethodGet, c.baseAddress.String(), nil)
	if err != nil {
		return err
	}

	if c.serverChallenge != nil {
		ec, err := c.PayloadManager.EncryptPlain(c.serverChallenge)
		if err != nil {
			return err
		}

		req.Header.Set(sessionChallenge, base58.Encode(ec))
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status not ok, received '%s'", resp.Status)
	}

	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return c.PayloadManager.Receive(payload)
}

func (c *PairingClient) getChallenge() error {
	c.baseAddress.Path = pairingChallenge
	resp, err := c.Get(c.baseAddress.String())
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status not ok, received '%s'", resp.Status)
	}

	c.serverChallenge, err = ioutil.ReadAll(resp.Body)
	return err
}

func StartUpPairingClient(db *multiaccounts.Database, cs, configJSON string) error {
	var conf PairingPayloadSourceConfig
	err := json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return err
	}

	ccp := new(ConnectionParams)
	err = ccp.FromString(cs)

	c, err := NewPairingClient(ccp, &PairingPayloadManagerConfig{db, conf})
	if err != nil {
		return err
	}

	return c.PairAccount()
}
