package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/sessions"

	"github.com/status-im/status-go/multiaccounts"
)

type PairingServer struct {
	Server
	PayloadManager

	pk   *ecdsa.PublicKey
	ek   []byte
	mode Mode

	cookieStore *sessions.CookieStore
}

type Config struct {
	// Connection fields
	PK       *ecdsa.PublicKey
	EK       []byte
	Cert     *tls.Certificate
	Hostname string
	Mode     Mode

	// Payload management fields
	*PairingPayloadManagerConfig
}

func makeCookieStore() (*sessions.CookieStore, error) {
	auth := make([]byte, 64)
	_, err := rand.Read(auth)
	if err != nil {
		return nil, err
	}

	enc := make([]byte, 32)
	_, err = rand.Read(enc)
	if err != nil {
		return nil, err
	}

	return sessions.NewCookieStore(auth, enc), nil
}

// NewPairingServer returns a *PairingServer init from the given *Config
func NewPairingServer(config *Config) (*PairingServer, error) {
	pm, err := NewPairingPayloadManager(config.EK, config.PairingPayloadManagerConfig)
	if err != nil {
		return nil, err
	}

	cs, err := makeCookieStore()
	if err != nil {
		return nil, err
	}

	return &PairingServer{Server: NewServer(
		config.Cert,
		config.Hostname,
		nil,
	),
		pk:             config.PK,
		ek:             config.EK,
		mode:           config.Mode,
		PayloadManager: pm,
		cookieStore:    cs,
	}, nil
}

// MakeConnectionParams generates a *ConnectionParams based on the Server's current state
func (s *PairingServer) MakeConnectionParams() (*ConnectionParams, error) {
	netIP := net.ParseIP(s.hostname)
	if netIP == nil {
		return nil, fmt.Errorf("invalid ip address given '%s'", s.hostname)
	}

	netIP4 := netIP.To4()
	if netIP4 != nil {
		netIP = netIP4
	}

	return NewConnectionParams(netIP, s.MustGetPort(), s.pk, s.ek, s.mode), nil
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
	s.SetHandlers(HandlerPatternMap{
		pairingReceive:   handlePairingReceive(s),
		pairingChallenge: handlePairingChallenge(s),
	})
	return s.Start()
}

func (s *PairingServer) startSendingAccountData() error {
	err := s.Mount()
	if err != nil {
		return err
	}

	s.SetHandlers(HandlerPatternMap{
		pairingSend:      challengeMiddleware(s, handlePairingSend(s)),
		pairingChallenge: handlePairingChallenge(s),
	})
	return s.Start()
}

// MakeFullPairingServer generates a fully configured and randomly seeded PairingServer
func MakeFullPairingServer(db *multiaccounts.Database, mode Mode, storeConfig PairingPayloadSourceConfig) (*PairingServer, error) {
	tlsKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	AESKey := make([]byte, 32)
	_, err = rand.Read(AESKey)
	if err != nil {
		return nil, err
	}

	outboundIP, err := GetOutboundIP()
	if err != nil {
		return nil, err
	}

	tlsCert, _, err := GenerateCertFromKey(tlsKey, time.Now(), outboundIP.String())
	if err != nil {
		return nil, err
	}

	return NewPairingServer(&Config{
		// Things that can be generated, and CANNOT come from the app client (well they could be this is better)
		PK:       &tlsKey.PublicKey,
		EK:       AESKey,
		Cert:     &tlsCert,
		Hostname: outboundIP.String(),

		// Things that can't be generated, but DO come from the app client
		Mode: mode,

		PairingPayloadManagerConfig: &PairingPayloadManagerConfig{
			// Things that can't be generated, but DO NOT come from app client
			DB: db,

			// Things that can't be generated, but DO come from the app client
			PairingPayloadSourceConfig: storeConfig,
		},
	})
}

// StartUpPairingServer generates a PairingServer, starts the pairing server in the correct mode
// and returns the ConnectionParams string to allow a PairingClient to make a successful connection.
func StartUpPairingServer(db *multiaccounts.Database, mode Mode, configJSON string) (string, error) {
	var conf PairingPayloadSourceConfig
	err := json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return "", err
	}

	ps, err := MakeFullPairingServer(db, mode, conf)
	if err != nil {
		return "", err
	}

	err = ps.StartPairing()
	if err != nil {
		return "", err
	}

	cp, err := ps.MakeConnectionParams()
	if err != nil {
		return "", err
	}

	return cp.ToString(), nil
}
