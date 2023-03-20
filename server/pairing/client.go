package pairing

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/signal"
)

/*
|--------------------------------------------------------------------------
| BaseClient
|--------------------------------------------------------------------------
|
|
|
*/

// BaseClient is responsible for lower level pairing.Client functionality common to dependent Client types
type BaseClient struct {
	*http.Client
	serverCert     *x509.Certificate
	baseAddress    *url.URL
	challengeTaker *ChallengeTaker
}

// NewBaseClient returns a fully qualified BaseClient from the given ConnectionParams
func NewBaseClient(c *ConnectionParams) (*BaseClient, error) {
	u, err := c.URL()
	if err != nil {
		return nil, err
	}

	serverCert, err := getServerCert(u)
	if err != nil {
		return nil, err
	}

	err = verifyCert(serverCert, c.publicKey)
	if err != nil {
		return nil, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCert.Raw})

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	if ok := rootCAs.AppendCertsFromPEM(certPem); !ok {
		return nil, fmt.Errorf("failed to append certPem to rootCAs")
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false, // MUST BE FALSE
			RootCAs:            rootCAs,
		},
	}

	cj, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &BaseClient{
		Client:         &http.Client{Transport: tr, Jar: cj},
		serverCert:     serverCert,
		challengeTaker: NewChallengeTaker(NewPayloadEncryptor(c.aesKey)),
		baseAddress:    u,
	}, nil
}

// getChallenge makes a call to the identified Server and receives a [32]byte challenge
func (c *BaseClient) getChallenge() error {
	c.baseAddress.Path = pairingChallenge
	resp, err := c.Get(c.baseAddress.String())
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[client] status not ok when getting challenge, received '%s'", resp.Status)
	}

	challenge, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	c.challengeTaker.SetChallenge(challenge)
	return nil
}

/*
|--------------------------------------------------------------------------
| SenderClient
|--------------------------------------------------------------------------
|
| With AccountPayloadMounter, RawMessagePayloadMounter and InstallationPayloadMounterReceiver
|
*/

// SenderClient is responsible for sending pairing data to a ReceiverServer
type SenderClient struct {
	*BaseClient
	accountMounter      PayloadMounter
	rawMessageMounter   *RawMessagePayloadMounter
	installationMounter *InstallationPayloadMounterReceiver
}

// NewSenderClient returns a fully qualified SenderClient created with the incoming parameters
func NewSenderClient(backend *api.GethStatusBackend, c *ConnectionParams, config *SenderClientConfig) (*SenderClient, error) {
	logger := logutils.ZapLogger().Named("SenderClient")
	pe := NewPayloadEncryptor(c.aesKey)

	bc, err := NewBaseClient(c)
	if err != nil {
		return nil, err
	}

	am, rmm, imr, err := NewPayloadMounters(logger, pe, backend, config.SenderConfig)
	if err != nil {
		return nil, err
	}

	return &SenderClient{
		BaseClient:          bc,
		accountMounter:      am,
		rawMessageMounter:   rmm,
		installationMounter: imr,
	}, nil
}

func (c *SenderClient) sendAccountData() error {
	err := c.accountMounter.Mount()
	if err != nil {
		return err
	}

	c.baseAddress.Path = pairingReceiveAccount
	resp, err := c.Post(c.baseAddress.String(), "application/octet-stream", bytes.NewBuffer(c.accountMounter.ToSend()))
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("[client] status not ok when sending account data, received '%s'", resp.Status)
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
		return err
	}

	signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingAccount})

	c.accountMounter.LockPayload()
	return nil
}

func (c *SenderClient) sendSyncDeviceData() error {
	err := c.rawMessageMounter.Mount()
	if err != nil {
		return err
	}

	c.baseAddress.Path = pairingReceiveSyncDevice
	resp, err := c.Post(c.baseAddress.String(), "application/octet-stream", bytes.NewBuffer(c.rawMessageMounter.ToSend()))
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("[client] status not okay when sending sync device data, status: %s", resp.Status)
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
		return err
	}

	signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionSyncDevice})
	return nil
}

func (c *SenderClient) receiveInstallationData() error {
	c.baseAddress.Path = pairingSendInstallation
	req, err := http.NewRequest(http.MethodGet, c.baseAddress.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("[client] status not ok when receiving installation data, received '%s'", resp.Status)
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
		return err
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
		return err
	}
	signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingInstallation})

	err = c.installationMounter.Receive(payload)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err.Error(), Action: ActionPairingInstallation})
		return err
	}
	signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess, Action: ActionPairingInstallation})
	return nil
}

// setupSendingClient creates a new SenderClient after parsing string inputs
func setupSendingClient(backend *api.GethStatusBackend, cs, configJSON string) (*SenderClient, error) {
	ccp := new(ConnectionParams)
	err := ccp.FromString(cs)
	if err != nil {
		return nil, err
	}

	conf := NewSenderClientConfig()
	err = json.Unmarshal([]byte(configJSON), conf)
	if err != nil {
		return nil, err
	}

	conf.SenderConfig.DB = backend.GetMultiaccountDB()

	return NewSenderClient(backend, ccp, conf)
}

// StartUpSendingClient creates a SenderClient and triggers all `send` calls in sequence to the ReceiverServer
func StartUpSendingClient(backend *api.GethStatusBackend, cs, configJSON string) error {
	c, err := setupSendingClient(backend, cs, configJSON)
	if err != nil {
		return err
	}
	err = c.sendAccountData()
	if err != nil {
		return err
	}
	err = c.sendSyncDeviceData()
	if err != nil {
		return err
	}
	return c.receiveInstallationData()
}

/*
|--------------------------------------------------------------------------
| ReceiverClient
|--------------------------------------------------------------------------
|
| With AccountPayloadReceiver, RawMessagePayloadReceiver, InstallationPayloadMounterReceiver
|
*/

// ReceiverClient is responsible for accepting pairing data to a SenderServer
type ReceiverClient struct {
	*BaseClient

	accountReceiver      PayloadReceiver
	rawMessageReceiver   *RawMessagePayloadReceiver
	installationReceiver *InstallationPayloadMounterReceiver
}

// NewReceiverClient returns a fully qualified ReceiverClient created with the incoming parameters
func NewReceiverClient(backend *api.GethStatusBackend, c *ConnectionParams, config *ReceiverClientConfig) (*ReceiverClient, error) {
	bc, err := NewBaseClient(c)
	if err != nil {
		return nil, err
	}

	logger := logutils.ZapLogger().Named("ReceiverClient")
	pe := NewPayloadEncryptor(c.aesKey)

	ar, rmr, imr, err := NewPayloadReceivers(logger, pe, backend, config.ReceiverConfig)
	if err != nil {
		return nil, err
	}

	return &ReceiverClient{
		BaseClient:           bc,
		accountReceiver:      ar,
		rawMessageReceiver:   rmr,
		installationReceiver: imr,
	}, nil
}

func (c *ReceiverClient) receiveAccountData() error {
	c.baseAddress.Path = pairingSendAccount
	req, err := http.NewRequest(http.MethodGet, c.baseAddress.String(), nil)
	if err != nil {
		return err
	}

	err = c.challengeTaker.DoChallenge(req)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("[client] status not ok when receiving account data, received '%s'", resp.Status)
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
		return err
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
		return err
	}
	signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingAccount})

	err = c.accountReceiver.Receive(payload)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err.Error(), Action: ActionPairingAccount})
		return err
	}
	signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess, Action: ActionPairingAccount})
	return nil
}

func (c *ReceiverClient) receiveSyncDeviceData() error {
	c.baseAddress.Path = pairingSendSyncDevice
	req, err := http.NewRequest(http.MethodGet, c.baseAddress.String(), nil)
	if err != nil {
		return err
	}

	err = c.challengeTaker.DoChallenge(req)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("[client] status not ok when receiving sync device data, received '%s'", resp.Status)
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
		return err
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
		return err
	}
	signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionSyncDevice})

	err = c.rawMessageReceiver.Receive(payload)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err.Error(), Action: ActionSyncDevice})
		return err
	}
	signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess, Action: ActionSyncDevice})
	return nil
}

func (c *ReceiverClient) sendInstallationData() error {
	err := c.installationReceiver.Mount()
	if err != nil {
		return err
	}

	c.baseAddress.Path = pairingReceiveInstallation
	req, err := http.NewRequest(http.MethodPost, c.baseAddress.String(), bytes.NewBuffer(c.installationReceiver.ToSend()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	err = c.challengeTaker.DoChallenge(req)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("[client] status not okay when sending installation data, status: %s", resp.Status)
		signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
		return err
	}

	signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingInstallation})
	return nil
}

// setupReceivingClient creates a new ReceiverClient after parsing string inputs
func setupReceivingClient(backend *api.GethStatusBackend, cs, configJSON string) (*ReceiverClient, error) {
	ccp := new(ConnectionParams)
	err := ccp.FromString(cs)
	if err != nil {
		return nil, err
	}

	conf := NewReceiverClientConfig()
	err = json.Unmarshal([]byte(configJSON), conf)
	if err != nil {
		return nil, err
	}

	conf.ReceiverConfig.DB = backend.GetMultiaccountDB()

	return NewReceiverClient(backend, ccp, conf)
}

// StartUpReceivingClient creates a ReceiverClient and triggers all `receive` calls in sequence to the SenderServer
func StartUpReceivingClient(backend *api.GethStatusBackend, cs, configJSON string) error {
	c, err := setupReceivingClient(backend, cs, configJSON)
	if err != nil {
		return err
	}
	err = c.getChallenge()
	if err != nil {
		return err
	}
	err = c.receiveAccountData()
	if err != nil {
		return err
	}
	err = c.receiveSyncDeviceData()
	if err != nil {
		return err
	}
	return c.sendInstallationData()
}
