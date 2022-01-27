package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

var (
	pf = "A Preferred Name"
)

func TestMessengerSyncSettings(t *testing.T) {
	suite.Run(t, new(MessengerSyncSettingsSuite))
}

type MessengerSyncSettingsSuite struct {
	suite.Suite
	bob   *Messenger
	alice *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerSyncSettingsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.bob = s.newMessenger()
	s.alice = s.newMessenger()
	_, err := s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *MessengerSyncSettingsSuite) TearDownTest() {
	s.Require().NoError(s.bob.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *MessengerSyncSettingsSuite) newMessengerWithOptions(shh types.Waku, privateKey *ecdsa.PrivateKey, options []Option) *Messenger {
	m, err := NewMessenger(
		"Test",
		privateKey,
		&testNode{shh: shh},
		uuid.New().String(),
		nil,
		options...,
	)
	s.Require().NoError(err)

	err = m.Init()
	s.Require().NoError(err)

	config := params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}

	networks := json.RawMessage("{}")
	setting := settings.Settings{
		Address:                   types.HexToAddress("0x1122334455667788990011223344556677889900"),
		AnonMetricsShouldSend:     false,
		Currency:                  "eth",
		CurrentNetwork:            "mainnet_rpc",
		DappsAddress:              types.HexToAddress("0x1122334455667788990011223344556677889900"),
		InstallationID:            "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:                    "0x1122334455667788990011223344556677889900",
		LatestDerivedPath:         0,
		Name:                      "Test",
		Networks:                  &networks,
		PhotoPath:                 "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreferredName:             &pf, // TODO this won't work, need to do a raw saveSettings call
		PreviewPrivacy:            false,
		PublicKey:                 "0x04112233445566778899001122334455667788990011223344556677889900112233445566778899001122334455667788990011223344556677889900",
		SigningPhrase:             "yurt joey vibe",
		SendPushNotifications:     true,
		ProfilePicturesVisibility: 1,
		DefaultSyncPeriod:         86400,
		UseMailservers:            true,
		LinkPreviewRequestEnabled: true,
		SendStatusUpdates:         true,
		WalletRootAddress:         types.HexToAddress("0x1122334455667788990011223344556677889900")}

	_ = m.settings.CreateSettings(setting, config)

	return m
}

func (s *MessengerSyncSettingsSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	options := []Option{
		WithCustomLogger(s.logger),
		WithDatabaseConfig(tmpFile.Name(), ""),
		WithDatasync(),
	}
	return s.newMessengerWithOptions(shh, privateKey, options)
}

func (s *MessengerSyncSettingsSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *MessengerSyncSettingsSuite) pairTwoDevices(device1, device2 *Messenger, deviceName, deviceType string) {
	// Send pairing data
	response, err := device1.SendPairInstallation(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Len(response.Chats(), 1)
	s.False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		device2,
		func(r *MessengerResponse) bool {
			for _, installation := range r.Installations {
				if installation.ID == device1.installationID {
					return installation.InstallationMetadata != nil && deviceName == installation.InstallationMetadata.Name && deviceType == installation.InstallationMetadata.DeviceType
				}
			}
			return false

		},
		"installation not received",
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Ensure installation is enabled
	err = device2.EnableInstallation(device1.installationID)
	s.Require().NoError(err)
}

func (s *MessengerSyncSettingsSuite) TestSyncSettings() {
	// Set Alice's installation metadata
	aim := &multidevice.InstallationMetadata{
		Name:       "alice's-device",
		DeviceType: "alice's-device-type",
	}
	err := s.alice.SetInstallationMetadata(s.alice.installationID, aim)
	s.Require().NoError(err)

	// Create Alice's other device
	alicesOtherDevice, err := newMessengerWithKey(s.shh, s.alice.identity, s.logger, nil)
	s.Require().NoError(err)

	im1 := &multidevice.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, im1)
	s.Require().NoError(err)

	// Pair alice's two devices
	s.pairTwoDevices(alicesOtherDevice, s.alice, im1.Name, im1.DeviceType)
	s.pairTwoDevices(s.alice, alicesOtherDevice, aim.Name, aim.DeviceType)

	// Check alice 1 settings values
	as, err := s.alice.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Equal("eth", as.Currency)
	s.Require().Equal(pf, as.PreferredName)
	s.Require().Exactly(settings.ProfilePicturesShowToContactsOnly, as.ProfilePicturesShowTo)
	s.Require().Exactly(settings.ProfilePicturesVisibilityContactsOnly, as.ProfilePicturesVisibility)

	// Check alice 2 settings values
	aos, err := alicesOtherDevice.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Equal("", aos.Currency)
	s.Require().Equal("", aos.PreferredName)
	s.Require().Exactly(settings.ProfilePicturesShowToContactsOnly, aos.ProfilePicturesShowTo)
	s.Require().Exactly(settings.ProfilePicturesVisibilityContactsOnly, aos.ProfilePicturesVisibility)

	// alice triggers global settings sync
	err = s.alice.syncSettings()
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		_, err = alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		ns, err := alicesOtherDevice.settings.GetSettings()
		if err != nil {
			return err
		}

		if ns.Currency != "eth" {
			return errors.New("settings sync not received")
		}
		return nil
	})
	s.Require().NoError(err)

	// Check alice 2 settings values
	aos, err = alicesOtherDevice.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Equal("eth", aos.Currency)

	// Alice 2 updated a setting which triggers the sync functionality
	err = alicesOtherDevice.settings.SaveSetting(settings.ProfilePicturesShowTo.GetReactName(), settings.ProfilePicturesShowToEveryone)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		_, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}

		ns, err := s.alice.settings.GetSettings()
		if err != nil {
			return err
		}

		if ns.ProfilePicturesShowTo != settings.ProfilePicturesShowToEveryone {
			return errors.New("settings sync not received")
		}
		return nil
	})
	s.Require().NoError(err)

	// Check alice 1 settings values
	as, err = s.alice.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Exactly(settings.ProfilePicturesShowToEveryone, as.ProfilePicturesShowTo)
}
