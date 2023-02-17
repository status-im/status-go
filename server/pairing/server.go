package pairing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/sessions"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/server"
)

type Server struct {
	server.Server
	PayloadManager
	rawMessagePayloadManager *RawMessagePayloadManager

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

	// AccountPayload management fields
	*AccountPayloadManagerConfig
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

// NewPairingServer returns a *Server init from the given *Config
func NewPairingServer(backend *api.GethStatusBackend, config *Config) (*Server, error) {
	logger := logutils.ZapLogger().Named("Server")
	accountPayloadManagerConfig := config.AccountPayloadManagerConfig
	pm, err := NewAccountPayloadManager(config.EK, accountPayloadManagerConfig, logger)
	if err != nil {
		return nil, err
	}

	cs, err := makeCookieStore()
	if err != nil {
		return nil, err
	}

	rmpm, err := NewRawMessagePayloadManager(logger, pm.accountPayload, config.EK, backend, accountPayloadManagerConfig.GetNodeConfig(), accountPayloadManagerConfig.GetSettingCurrentNetwork())
	if err != nil {
		return nil, err
	}

	s := &Server{Server: server.NewServer(
		config.Cert,
		config.Hostname,
		nil,
		logger,
	),
		pk:                       config.PK,
		ek:                       config.EK,
		mode:                     config.Mode,
		PayloadManager:           pm,
		cookieStore:              cs,
		rawMessagePayloadManager: rmpm,
	}
	s.SetTimeout(config.GetTimeout())

	return s, nil
}

// MakeConnectionParams generates a *ConnectionParams based on the Server's current state
func (s *Server) MakeConnectionParams() (*ConnectionParams, error) {
	hostname := s.GetHostname()
	netIP := net.ParseIP(hostname)
	if netIP == nil {
		return nil, fmt.Errorf("invalid ip address given '%s'", hostname)
	}

	netIP4 := netIP.To4()
	if netIP4 != nil {
		netIP = netIP4
	}

	return NewConnectionParams(netIP, s.MustGetPort(), s.pk, s.ek, s.mode), nil
}

func (s *Server) StartPairing() error {
	switch s.mode {
	case Receiving:
		return s.startReceivingData()
	case Sending:
		return s.startSendingData()
	default:
		return fmt.Errorf("invalid server mode '%d'", s.mode)
	}
}

func (s *Server) startReceivingData() error {
	s.SetHandlers(server.HandlerPatternMap{
		pairingReceiveAccount:    handlePairingReceive(s),
		pairingChallenge:         handlePairingChallenge(s),
		pairingSyncDeviceReceive: handleParingSyncDeviceReceive(s),
	})
	return s.Start()
}

func (s *Server) startSendingData() error {
	err := s.Mount()
	if err != nil {
		return err
	}

	s.SetHandlers(server.HandlerPatternMap{
		pairingSendAccount:    challengeMiddleware(s, handlePairingSend(s)),
		pairingChallenge:      handlePairingChallenge(s),
		pairingSyncDeviceSend: challengeMiddleware(s, handlePairingSyncDeviceSend(s)),
	})
	return s.Start()
}

// MakeFullPairingServer generates a fully configured and randomly seeded Server
func MakeFullPairingServer(backend *api.GethStatusBackend, mode Mode, storeConfig *PayloadSourceConfig) (*Server, error) {
	tlsKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	AESKey := make([]byte, 32)
	_, err = rand.Read(AESKey)
	if err != nil {
		return nil, err
	}

	outboundIP, err := server.GetOutboundIP()
	if err != nil {
		return nil, err
	}

	tlsCert, _, err := GenerateCertFromKey(tlsKey, time.Now(), outboundIP.String())
	if err != nil {
		return nil, err
	}
	return NewPairingServer(backend, &Config{
		// Things that can be generated, and CANNOT come from the app client (well they could be this is better)
		PK:       &tlsKey.PublicKey,
		EK:       AESKey,
		Cert:     &tlsCert,
		Hostname: outboundIP.String(),

		// Things that can't be generated, but DO come from the app client
		Mode: mode,

		AccountPayloadManagerConfig: &AccountPayloadManagerConfig{
			// Things that can't be generated, but DO NOT come from app client
			DB: backend.GetMultiaccountDB(),

			// Things that can't be generated, but DO come from the app client
			PayloadSourceConfig: storeConfig,
		},
	})
}

// StartUpPairingServer generates a Server, starts the pairing server in the correct mode
// and returns the ConnectionParams string to allow a Client to make a successful connection.
func StartUpPairingServer(backend *api.GethStatusBackend, mode Mode, configJSON string) (string, error) {
	conf, err := NewPayloadSourceForServer(configJSON, mode)
	if err != nil {
		return "", err
	}

	ps, err := MakeFullPairingServer(backend, mode, conf)
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
