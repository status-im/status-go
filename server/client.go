package server

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	*http.Client

	baseAddress *url.URL
	certPEM     []byte
	privateKey  *ecdsa.PrivateKey
}

func NewClient(c *ConnectionParams) (*Client, error) {
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

	return &Client{
		Client:      &http.Client{Transport: tr},
		baseAddress: u,
		certPEM:     certPem,
		privateKey:  c.privateKey,
	}, nil
}
