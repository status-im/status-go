package pairing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"go.uber.org/zap"

	"github.com/status-im/status-go/timesource"

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

	config ServerConfig
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
			config.ListenIP.String(),
			nil,
			logger,
		),
		challengeGiver: cg,
		config:         *config,
	}
	bs.SetTimeout(config.Timeout)
	return bs, nil
}

// MakeConnectionParams generates a *ConnectionParams based on the Server's current state
func (s *BaseServer) MakeConnectionParams() (*ConnectionParams, error) {
	return NewConnectionParams(s.config.IPAddresses, s.MustGetPort(), s.config.PK, s.config.EK), nil
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

	ips, err := server.GetLocalAddressesForPairingServer()
	if err != nil {
		return err
	}

	now, err := timesource.GetCurrentTime()
	if err != nil {
		return err
	}
	log.Debug("pairing server generate cert", "system time", time.Now().String(), "timesource time", now.String())
	tlsCert, _, err := GenerateCertFromKey(tlsKey, *now, ips, []string{})
	if err != nil {
		return err
	}

	config.PK = &tlsKey.PublicKey
	config.EK = AESKey
	config.Cert = &tlsCert
	config.IPAddresses = ips
	config.ListenIP = net.IPv4zero

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
	rawMessageMounter   PayloadMounter
	installationMounter PayloadMounterReceiver
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
		pairingSendAccount:    middlewareChallenge(s.challengeGiver, handleSendAccount(s.GetLogger(), s.accountMounter)),
		pairingSendSyncDevice: middlewareChallenge(s.challengeGiver, handlePairingSyncDeviceSend(s.GetLogger(), s.rawMessageMounter)),
		// TODO implement refactor of installation data exchange to follow the send/receive pattern of
		//  the other handlers.
		//  https://github.com/status-im/status-go/issues/3304
		// receive installation data from receiver
		pairingReceiveInstallation: middlewareChallenge(s.challengeGiver, handleReceiveInstallation(s.GetLogger(), s.installationMounter)),
	})
	return s.Start()
}

// MakeFullSenderServer generates a fully configured and randomly seeded SenderServer
func MakeFullSenderServer(backend *api.GethStatusBackend, config *SenderServerConfig) (*SenderServer, error) {
	err := MakeServerConfig(config.ServerConfig)
	if err != nil {
		return nil, err
	}

	config.SenderConfig.DB = backend.GetMultiaccountDB()
	return NewSenderServer(backend, config)
}

// StartUpSenderServer generates a SenderServer, starts the sending server
// and returns the ConnectionParams string to allow a ReceiverClient to make a successful connection.
func StartUpSenderServer(backend *api.GethStatusBackend, configJSON string) (string, error) {
	conf := NewSenderServerConfig()
	err := json.Unmarshal([]byte(configJSON), conf)
	if err != nil {
		return "", err
	}
	if len(conf.SenderConfig.ChatKey) == 0 {
		err = validateAndVerifyPassword(conf, conf.SenderConfig)
		if err != nil {
			return "", err
		}
	}

	ps, err := MakeFullSenderServer(backend, conf)
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
		pairingReceiveAccount:    handleReceiveAccount(s.GetLogger(), s.accountReceiver),
		pairingReceiveSyncDevice: handleParingSyncDeviceReceive(s.GetLogger(), s.rawMessageReceiver),
		// TODO implement refactor of installation data exchange to follow the send/receive pattern of
		//  the other handlers.
		//  https://github.com/status-im/status-go/issues/3304
		// send installation data back to sender
		pairingSendInstallation: middlewareChallenge(s.challengeGiver, handleSendInstallation(s.GetLogger(), s.installationReceiver)),
	})
	return s.Start()
}

// MakeFullReceiverServer generates a fully configured and randomly seeded ReceiverServer
func MakeFullReceiverServer(backend *api.GethStatusBackend, config *ReceiverServerConfig) (*ReceiverServer, error) {
	err := MakeServerConfig(config.ServerConfig)
	if err != nil {
		return nil, err
	}

	// ignore err because we allow no active account here
	activeAccount, _ := backend.GetActiveAccount()
	if activeAccount != nil {
		config.ReceiverConfig.LoggedInKeyUID = activeAccount.KeyUID
	}
	config.ReceiverConfig.DB = backend.GetMultiaccountDB()

	return NewReceiverServer(backend, config)
}

// StartUpReceiverServer generates a ReceiverServer, starts the sending server
// and returns the ConnectionParams string to allow a SenderClient to make a successful connection.
func StartUpReceiverServer(backend *api.GethStatusBackend, configJSON string) (string, error) {
	conf := NewReceiverServerConfig()
	err := json.Unmarshal([]byte(configJSON), conf)
	if err != nil {
		return "", err
	}
	err = validateAndVerifyNodeConfig(conf, conf.ReceiverConfig)
	if err != nil {
		return "", err
	}

	ps, err := MakeFullReceiverServer(backend, conf)
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

/*
|--------------------------------------------------------------------------
| type KeystoreFilesSenderServer struct {
|--------------------------------------------------------------------------
*/

type KeystoreFilesSenderServer struct {
	*BaseServer
	keystoreFilesMounter PayloadMounter
}

func NewKeystoreFilesSenderServer(backend *api.GethStatusBackend, config *KeystoreFilesSenderServerConfig) (*KeystoreFilesSenderServer, error) {
	logger := logutils.ZapLogger().Named("SenderServer")
	e := NewPayloadEncryptor(config.ServerConfig.EK)

	bs, err := NewBaseServer(logger, e, config.ServerConfig)
	if err != nil {
		return nil, err
	}

	kfm, err := NewKeystoreFilesPayloadMounter(backend, e, config.SenderConfig, logger)
	if err != nil {
		return nil, err
	}

	return &KeystoreFilesSenderServer{
		BaseServer:           bs,
		keystoreFilesMounter: kfm,
	}, nil
}

func (s *KeystoreFilesSenderServer) startSendingData() error {
	s.SetHandlers(server.HandlerPatternMap{
		pairingChallenge:   handlePairingChallenge(s.challengeGiver),
		pairingSendAccount: middlewareChallenge(s.challengeGiver, handleSendAccount(s.GetLogger(), s.keystoreFilesMounter)),
	})
	return s.Start()
}

// MakeFullSenderServer generates a fully configured and randomly seeded KeystoreFilesSenderServer
func MakeKeystoreFilesSenderServer(backend *api.GethStatusBackend, config *KeystoreFilesSenderServerConfig) (*KeystoreFilesSenderServer, error) {
	err := MakeServerConfig(config.ServerConfig)
	if err != nil {
		return nil, err
	}

	return NewKeystoreFilesSenderServer(backend, config)
}

// StartUpKeystoreFilesSenderServer generates a KeystoreFilesSenderServer, starts the sending server
// and returns the ConnectionParams string to allow a ReceiverClient to make a successful connection.
func StartUpKeystoreFilesSenderServer(backend *api.GethStatusBackend, configJSON string) (string, error) {
	conf := NewKeystoreFilesSenderServerConfig()
	err := json.Unmarshal([]byte(configJSON), conf)
	if err != nil {
		return "", err
	}

	err = validateKeystoreFilesConfig(backend, conf)
	if err != nil {
		return "", err
	}

	ps, err := MakeKeystoreFilesSenderServer(backend, conf)
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
