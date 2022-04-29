package server

import (
	"crypto/ecdsa"
	"crypto/tls"
)

type PairingServer struct {
	Server

	pk       *ecdsa.PrivateKey
}

type Config struct {
	PK    *ecdsa.PrivateKey
	Cert     *tls.Certificate
	Hostname string
}

// NewPairingServer returns a *NewPairingServer init from the given *Config
func NewPairingServer(config *Config) *PairingServer {
	return &PairingServer{Server: NewServer(
		config.Cert,
		config.Hostname,
	),
	pk: config.PK}
}
