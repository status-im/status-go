package pairing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/server"
)

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
	challengeGiver *ChallengeGiver

	pk *ecdsa.PublicKey
	ek []byte
	// TODO remove mode from pairing process
	//  https://github.com/status-im/status-go/issues/3301
	mode Mode
}

// NewBaseServer returns a *BaseServer init from the given *SenderServerConfig
func NewBaseServer(logger *zap.Logger, e *PayloadEncryptor, config *ServerConfig) (*BaseServer, error) {
	cg, err := NewChallengeGiver(e, logger)
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
		challengeGiver: cg,
		pk:             config.PK,
		ek:             config.EK,
		mode:           config.Mode,
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
	e := NewPayloadEncryptor(config.ServerConfig.EK)

	bs, err := NewBaseServer(logger, e, config.ServerConfig)
	if err != nil {
		return nil, err
	}

	am, rmm, imr, err := NewPayloadMounters(logger, e, backend, config.SenderConfig)
	if err != nil {
		return nil, err
	}

	return &SenderServer{
		BaseServer:          bs,
		accountMounter:      am,
		rawMessageMounter:   rmm,
		installationMounter: imr,
	}, nil
}

func (s *SenderServer) startSendingData() error {
	s.SetHandlers(server.HandlerPatternMap{
		pairingChallenge:      handlePairingChallenge(s.challengeGiver),
		pairingSendAccount:    middlewareChallenge(s.challengeGiver, handleSendAccount(s, s.accountMounter)),
		pairingSendSyncDevice: middlewareChallenge(s.challengeGiver, handlePairingSyncDeviceSend(s, s.rawMessageMounter)),
		// TODO implement refactor of installation data exchange to follow the send/receive pattern of
		//  the other handlers.
		//  https://github.com/status-im/status-go/issues/3304
		// receive installation data from receiver
		pairingReceiveInstallation: middlewareChallenge(s.challengeGiver, handleReceiveInstallation(s, s.installationMounter)),
	})
	return s.Start()
}

// MakeFullSenderServer generates a fully configured and randomly seeded SenderServer
func MakeFullSenderServer(backend *api.GethStatusBackend, mode Mode, config *SenderServerConfig) (*SenderServer, error) {
	err := MakeServerConfig(config.ServerConfig)
	if err != nil {
		return nil, err
	}

	config.SenderConfig.DB = backend.GetMultiaccountDB()
	return NewSenderServer(backend, config)
}

// StartUpSenderServer generates a SenderServer, starts the sending server in the correct mode
// and returns the ConnectionParams string to allow a ReceiverClient to make a successful connection.
func StartUpSenderServer(backend *api.GethStatusBackend, mode Mode, configJSON string) (string, error) {
	conf := NewSenderServerConfig()
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
	e := NewPayloadEncryptor(config.ServerConfig.EK)

	bs, err := NewBaseServer(logger, e, config.ServerConfig)
	if err != nil {
		return nil, err
	}

	ar, rmr, imr, err := NewPayloadReceivers(logger, e, backend, config.ReceiverConfig)
	if err != nil {
		return nil, err
	}

	return &ReceiverServer{
		BaseServer:           bs,
		accountReceiver:      ar,
		rawMessageReceiver:   rmr,
		installationReceiver: imr,
	}, nil
}

func (s *ReceiverServer) startReceivingData() error {
	s.SetHandlers(server.HandlerPatternMap{
		pairingChallenge:         handlePairingChallenge(s.challengeGiver),
		pairingReceiveAccount:    handleReceiveAccount(s, s.accountReceiver),
		pairingReceiveSyncDevice: handleParingSyncDeviceReceive(s, s.rawMessageReceiver),
		// TODO implement refactor of installation data exchange to follow the send/receive pattern of
		//  the other handlers.
		//  https://github.com/status-im/status-go/issues/3304
		// send installation data back to sender
		pairingSendInstallation: middlewareChallenge(s.challengeGiver, handleSendInstallation(s, s.installationReceiver)),
	})
	return s.Start()
}

// MakeFullReceiverServer generates a fully configured and randomly seeded ReceiverServer
func MakeFullReceiverServer(backend *api.GethStatusBackend, mode Mode, config *ReceiverServerConfig) (*ReceiverServer, error) {
	err := MakeServerConfig(config.ServerConfig)
	if err != nil {
		return nil, err
	}

	activeAccount, _ := backend.GetActiveAccount()
	if activeAccount != nil {
		config.ReceiverConfig.LoggedInKeyUID = activeAccount.KeyUID
	}
	config.ReceiverConfig.DB = backend.GetMultiaccountDB()

	return NewReceiverServer(backend, config)
}

// StartUpReceiverServer generates a ReceiverServer, starts the sending server in the correct mode
// and returns the ConnectionParams string to allow a SenderClient to make a successful connection.
func StartUpReceiverServer(backend *api.GethStatusBackend, mode Mode, configJSON string) (string, error) {
	conf := NewReceiverServerConfig()
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
