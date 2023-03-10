package pairing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/sessions"
	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/server"
)

type Server struct {
	server.Server
	PayloadManager
	rawMessagePayloadManager   *RawMessagePayloadManager
	installationPayloadManager *InstallationPayloadManager

	pk   *ecdsa.PublicKey
	ek   []byte
	mode Mode

	cookieStore *sessions.CookieStore
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

	rmpm, err := NewRawMessagePayloadManager(logger, pm.accountPayload, config.EK, backend, accountPayloadManagerConfig.GetNodeConfig(), accountPayloadManagerConfig.GetSettingCurrentNetwork(), accountPayloadManagerConfig.GetDeviceType())
	if err != nil {
		return nil, err
	}

	ipm, err := NewInstallationPayloadManager(logger, config.EK, backend, accountPayloadManagerConfig.GetDeviceType())
	if err != nil {
		return nil, err
	}

	cs, err := makeCookieStore()
	if err != nil {
		return nil, err
	}

	s := &Server{Server: server.NewServer(
		config.Cert,
		config.Hostname,
		nil,
		logger,
	),
		pk:                         config.PK,
		ek:                         config.EK,
		mode:                       config.Mode,
		PayloadManager:             pm,
		cookieStore:                cs,
		rawMessagePayloadManager:   rmpm,
		installationPayloadManager: ipm,
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
		pairingChallenge:         handlePairingChallenge(s),
		pairingReceiveAccount:    handleReceiveAccount(s),
		pairingReceiveSyncDevice: handleParingSyncDeviceReceive(s),
		// TODO implement refactor of installation data exchange to follow the send/receive pattern of
		//  the other handlers.
		// send installation data back to sender
		pairingSendInstallation: handleSendInstallation(s),
	})
	return s.Start()
}

func (s *Server) startSendingData() error {
	err := s.Mount()
	if err != nil {
		return err
	}

	s.SetHandlers(server.HandlerPatternMap{
		pairingChallenge:      handlePairingChallenge(s),
		pairingSendAccount:    challengeMiddleware(s, handleSendAccount(s)),
		pairingSendSyncDevice: challengeMiddleware(s, handlePairingSyncDeviceSend(s)),
		// TODO implement refactor of installation data exchange to follow the send/receive pattern of
		//  the other handlers.
		// receive installation data from receiver
		pairingReceiveInstallation: challengeMiddleware(s, handleReceiveInstallation(s)),
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

	accountPayloadManagerConfig := &AccountPayloadManagerConfig{
		// Things that can't be generated, but DO NOT come from app client
		DB: backend.GetMultiaccountDB(),

		// Things that can't be generated, but DO come from the app client
		PayloadSourceConfig: storeConfig,
	}
	if mode == Receiving {
		updateLoggedInKeyUID(accountPayloadManagerConfig, backend)
	}

	return NewPairingServer(backend, &Config{
		// Things that can be generated, and CANNOT come from the app client (well they could be this is better)
		PK:       &tlsKey.PublicKey,
		EK:       AESKey,
		Cert:     &tlsCert,
		Hostname: outboundIP.String(),

		// Things that can't be generated, but DO come from the app client
		Mode: mode,

		AccountPayloadManagerConfig: accountPayloadManagerConfig,
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

/*
|--------------------------------------------------------------------------
| type BaseServer struct {
|--------------------------------------------------------------------------
|
|
|
*/

type BaseServer struct {
	server.Server

	cookieStore *sessions.CookieStore

	pk   *ecdsa.PublicKey
	ek   []byte
	mode Mode
}

// NewBaseServer returns a *BaseServer init from the given *SenderServerConfig
func NewBaseServer(logger *zap.Logger, config *ServerConfig) (*BaseServer, error) {
	cs, err := makeCookieStore()
	if err != nil {
		return nil, err
	}

	bs := &BaseServer{
		Server: server.NewServer(
			config.Cert,
			config.Hostname,
			nil,
			logger,
		),
		cookieStore: cs,
		pk:          config.PK,
		ek:          config.EK,
		mode:        config.Mode,
	}
	bs.SetTimeout(config.Timeout)
	return bs, nil
}

// MakeConnectionParams generates a *ConnectionParams based on the Server's current state
func (s *BaseServer) MakeConnectionParams() (*ConnectionParams, error) {
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

func (s *BaseServer) GetCookieStore() *sessions.CookieStore {
	return s.cookieStore
}

func (s *BaseServer) DecryptPlain(data []byte) ([]byte, error) {
	return common.Decrypt(data, s.ek)
}

func MakeServerConfig(config *ServerConfig) error {
	tlsKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	AESKey := make([]byte, 32)
	_, err = rand.Read(AESKey)
	if err != nil {
		return err
	}

	outboundIP, err := server.GetOutboundIP()
	if err != nil {
		return err
	}

	tlsCert, _, err := GenerateCertFromKey(tlsKey, time.Now(), outboundIP.String())
	if err != nil {
		return err
	}

	config.PK = &tlsKey.PublicKey
	config.EK = AESKey
	config.Cert = &tlsCert
	config.Hostname = outboundIP.String()
	return nil
}

/*
|--------------------------------------------------------------------------
| type SenderServer struct {
|--------------------------------------------------------------------------
|
| With AccountPayloadMounter, RawMessagePayloadMounter and InstallationPayloadMounterReceiver
|
*/

type SenderServer struct {
	*BaseServer
	accountMounter      PayloadMounter
	rawMessageMounter   *RawMessagePayloadMounter
	installationMounter *InstallationPayloadMounterReceiver
}

// NewSenderServer returns a *SenderServer init from the given *SenderServerConfig
func NewSenderServer(backend *api.GethStatusBackend, config *SenderServerConfig) (*SenderServer, error) {
	logger := logutils.ZapLogger().Named("SenderServer")
	e := NewPayloadEncryptor(config.Server.EK)

	bs, err := NewBaseServer(logger, config.Server)
	if err != nil {
		return nil, err
	}

	pm, err := NewAccountPayloadMounter(e, config.Sender, logger)
	if err != nil {
		return nil, err
	}
	rmpm := NewRawMessagePayloadMounter(logger, e, backend, config.Sender)
	ipm := NewInstallationPayloadMounterReceiver(logger, e, backend, config.Sender.DeviceType)

	return &SenderServer{
		BaseServer:          bs,
		accountMounter:      pm,
		rawMessageMounter:   rmpm,
		installationMounter: ipm,
	}, nil
}

func (s *SenderServer) startSendingData() error {
	s.SetHandlers(server.HandlerPatternMap{
		pairingChallenge:      handlePairingChallenge(s),
		pairingSendAccount:    challengeMiddleware(s, handleSendAccount(s, s.accountMounter)),
		pairingSendSyncDevice: challengeMiddleware(s, handlePairingSyncDeviceSend(s, s.rawMessageMounter)),
		// TODO implement refactor of installation data exchange to follow the send/receive pattern of
		//  the other handlers.
		// receive installation data from receiver
		pairingReceiveInstallation: challengeMiddleware(s, handleReceiveInstallation(s, s.installationMounter)),
	})
	return s.Start()
}

// MakeFullSenderServer generates a fully configured and randomly seeded SenderServer
func MakeFullSenderServer(backend *api.GethStatusBackend, mode Mode, config *SenderServerConfig) (*SenderServer, error) {
	err := MakeServerConfig(config.Server)
	if err != nil {
		return nil, err
	}

	config.Sender.DB = backend.GetMultiaccountDB()
	return NewSenderServer(backend, config)
}

// StartUpSenderServer generates a SenderServer, starts the sending server in the correct mode
// and returns the ConnectionParams string to allow a ReceiverClient to make a successful connection.
func StartUpSenderServer(backend *api.GethStatusBackend, mode Mode, configJSON string) (string, error) {
	conf := new(SenderServerConfig)
	err := json.Unmarshal([]byte(configJSON), conf)
	if err != nil {
		return "", err
	}

	ps, err := MakeFullSenderServer(backend, mode, conf)
	if err != nil {
		return "", err
	}

	err = ps.startSendingData()
	if err != nil {
		return "", err
	}

	cp, err := ps.MakeConnectionParams()
	if err != nil {
		return "", err
	}

	return cp.ToString(), nil
}

/*
|--------------------------------------------------------------------------
| ReceiverServer
|--------------------------------------------------------------------------
|
| With AccountPayloadReceiver, RawMessagePayloadReceiver, InstallationPayloadMounterReceiver
|
*/

type ReceiverServer struct {
	*BaseServer
	accountReceiver      PayloadReceiver
	rawMessageReceiver   PayloadReceiver
	installationReceiver PayloadMounterReceiver
}

// NewReceiverServer returns a *SenderServer init from the given *ReceiverServerConfig
func NewReceiverServer(backend *api.GethStatusBackend, config *ReceiverServerConfig) (*ReceiverServer, error) {
	logger := logutils.ZapLogger().Named("SenderServer")
	e := NewPayloadEncryptor(config.Server.EK)

	bs, err := NewBaseServer(logger, config.Server)
	if err != nil {
		return nil, err
	}

	ar, err := NewAccountPayloadReceiver(e, config.Receiver, logger)
	if err != nil {
		return nil, err
	}
	rmr := NewRawMessagePayloadReceiver(logger, ar.accountPayload, e, backend, config.Receiver)
	imr := NewInstallationPayloadMounterReceiver(logger, e, backend, config.Receiver.DeviceType)

	return &ReceiverServer{
		BaseServer:           bs,
		accountReceiver:      ar,
		rawMessageReceiver:   rmr,
		installationReceiver: imr,
	}, nil
}

func (s *ReceiverServer) startReceivingData() error {
	s.SetHandlers(server.HandlerPatternMap{
		pairingChallenge:         handlePairingChallenge(s),
		pairingReceiveAccount:    handleReceiveAccount(s, s.accountReceiver),
		pairingReceiveSyncDevice: handleParingSyncDeviceReceive(s, s.rawMessageReceiver),
		// TODO implement refactor of installation data exchange to follow the send/receive pattern of
		//  the other handlers.
		// send installation data back to sender
		pairingSendInstallation: handleSendInstallation(s, s.installationReceiver),
	})
	return s.Start()
}

// MakeFullReceiverServer generates a fully configured and randomly seeded ReceiverServer
func MakeFullReceiverServer(backend *api.GethStatusBackend, mode Mode, config *ReceiverServerConfig) (*ReceiverServer, error) {
	err := MakeServerConfig(config.Server)
	if err != nil {
		return nil, err
	}

	activeAccount, _ := backend.GetActiveAccount()
	if activeAccount != nil {
		config.Receiver.LoggedInKeyUID = activeAccount.KeyUID
	}
	config.Receiver.DB = backend.GetMultiaccountDB()

	return NewReceiverServer(backend, config)
}

// StartUpReceiverServer generates a ReceiverServer, starts the sending server in the correct mode
// and returns the ConnectionParams string to allow a SenderClient to make a successful connection.
func StartUpReceiverServer(backend *api.GethStatusBackend, mode Mode, configJSON string) (string, error) {
	conf := new(ReceiverServerConfig)
	err := json.Unmarshal([]byte(configJSON), conf)
	if err != nil {
		return "", err
	}

	ps, err := MakeFullReceiverServer(backend, mode, conf)
	if err != nil {
		return "", err
	}

	err = ps.startReceivingData()
	if err != nil {
		return "", err
	}

	cp, err := ps.MakeConnectionParams()
	if err != nil {
		return "", err
	}

	return cp.ToString(), nil
}
