package server

import (
	"crypto/ecdsa"
	"crypto/tls"
	"fmt"
	"net"
)

type PairingServer struct {
	Server

	pk      *ecdsa.PrivateKey
	mode    Mode
	payload *PayloadManager
}

type Config struct {
	PK       *ecdsa.PrivateKey
	Cert     *tls.Certificate
	Hostname string
	Mode     Mode
}

// NewPairingServer returns a *PairingServer init from the given *Config
func NewPairingServer(config *Config) (*PairingServer, error) {
	pm, err := NewPayloadManager(config.PK)
	if err != nil {
		return nil, err
	}

	return &PairingServer{Server: NewServer(
		config.Cert,
		config.Hostname,
	),
		pk:      config.PK,
		mode:    config.Mode,
		payload: pm}, nil
}

// MakeConnectionParams generates a *ConnectionParams based on the Server's current state
func (s *PairingServer) MakeConnectionParams() (*ConnectionParams, error) {
	switch {
	case s.cert == nil:
		return nil, fmt.Errorf("server has no cert set")
	case s.cert.Leaf == nil:
		return nil, fmt.Errorf("server cert has no Leaf set")
	case s.cert.Leaf.NotBefore.IsZero():
		return nil, fmt.Errorf("server cert Leaf has a zero value NotBefore")
	}

	netIP := net.ParseIP(s.hostname)
	if netIP == nil {
		return nil, fmt.Errorf("invalid ip address given '%s'", s.hostname)
	}

	netIP4 := netIP.To4()
	if netIP4 != nil {
		netIP = netIP4
	}

	if s.port == 0 {
		return nil, fmt.Errorf("port is 0, listener is not yet set")
	}

	return NewConnectionParams(netIP, s.port, s.pk, s.cert.Leaf.NotBefore, s.mode), nil
}

func (s *PairingServer) MountPayload(data []byte) error {
	return s.payload.Mount(data)
}

func (s *PairingServer) StartPairing() error {
	switch s.mode {
	case Receiving:
		return s.startReceivingAccountData()
	case Sending:
		return s.startSendingAccountData()
	default:
		return fmt.Errorf("invalid server mode '%d'", s.mode)
	}
}

func (s *PairingServer) startReceivingAccountData() error {
	s.SetHandlers(HandlerPatternMap{pairingReceive: handlePairingReceive(s)})
	return s.Start()
}

func (s *PairingServer) startSendingAccountData() error {
	s.SetHandlers(HandlerPatternMap{pairingSend: handlePairingSend(s)})
	return s.Start()
}
