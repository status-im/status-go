package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/status-im/status-go/multiaccounts"
	"net"
	"time"

	"github.com/gorilla/sessions"
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

	if s.port == 0 {
		return nil, fmt.Errorf("port is 0, listener is not yet set")
	}

	return NewConnectionParams(netIP, s.port, s.pk, s.ek, s.mode), nil
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
	s.SetHandlers(HandlerPatternMap{
		pairingSend:      challengeMiddleware(s, handlePairingSend(s)),
		pairingChallenge: handlePairingChallenge(s),
	})
	return s.Start()
}

func MakeFullPairingServer(db *multiaccounts.Database, mode Mode, keystorePath, keyUID, password string) (*PairingServer, error) {
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
		// Things that can be generated
		PK:       &tlsKey.PublicKey,
		EK:       AESKey,
		Cert:     &tlsCert,
		Hostname: outboundIP.String(),

		// Things that can't be generated, but do come from the client
		Mode: mode,

		PairingPayloadManagerConfig: &PairingPayloadManagerConfig{
			// Things that can't be generated, but can't come from client
			DB: db,

			// Things that can't be generated, but do come from the client
			KeystorePath: keystorePath,
			KeyUID:       keyUID,
			Password:     password,
		},
	})
}
