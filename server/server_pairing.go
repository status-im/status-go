package server

import (
	"crypto/tls"
)

type PairingServer struct {
	Server
}

type Config struct {
	Cert     *tls.Certificate
	Hostname string
}

// NewPairingServer returns a *NewPairingServer init from the given *Config
func NewPairingServer(config *Config) *PairingServer {
	return &PairingServer{Server: NewServer(
		config.Cert,
		config.Hostname,
	)}
}
