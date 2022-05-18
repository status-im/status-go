package server

import (
	"crypto/tls"
	"net"
)

type PairingServer struct {
	Server
}

type Config struct {
	Cert  *tls.Certificate
	NetIP net.IP
}

// NewPairingServer returns a *NewPairingServer init from the given *Config
func NewPairingServer(config *Config) *PairingServer {
	return &PairingServer{Server: NewServer(
		config.Cert,
		config.NetIP,
	)}
}
