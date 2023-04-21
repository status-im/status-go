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
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/stickers"
	"github.com/status-im/status-go/waku"
)

var (
	pf         = "A Preferred Name"
	pf2        = "AnotherPreferredName.eth"
	rawSticker = []byte(`{
  "1":{
    "author":"cryptoworld1373",
    "id":1,
    "name":"Status Cat",
    "preview":"e3010170122050efc0a3e661339f31e1e44b3d15a1bf4e501c965a0523f57b701667fa90ccca",
    "price":0,
    "stickers":[
      {"hash":"e30101701220eab9a8ef4eac6c3e5836a3768d8e04935c10c67d9a700436a0e53199e9b64d29"},
      {"hash":"e30101701220c8f28aebe4dbbcee896d1cdff89ceeaceaf9f837df55c79125388f954ee5f1fe"},
      {"hash":"e301017012204861f93e29dd8e7cf6699135c7b13af1bce8ceeaa1d9959ab8592aa20f05d15f"},
      {"hash":"e301017012203ffa57a51cceaf2ce040852de3b300d395d5ba4d70e08ba993f93a25a387e3a9"},
      {"hash":"e301017012204f2674db0bc7f7cfc0382d1d7f79b4ff73c41f5c487ef4c3bb3f3a4cf3f87d70"},
      {"hash":"e30101701220e8d4d8b9fb5f805add2f63c1cb5c891e60f9929fc404e3bb725aa81628b97b5f"},
      {"hash":"e301017012206fdad56fe7a2facb02dabe8294f3ac051443fcc52d67c2fbd8615eb72f9d74bd"},
      {"hash":"e30101701220a691193cf0559905c10a3c5affb9855d730eae05509d503d71327e6c820aaf98"},
      {"hash":"e30101701220d8004af925f8e85b4e24813eaa5ef943fa6a0c76035491b64fbd2e632a5cc2fd"},
      {"hash":"e3010170122049f7bc650615568f14ee1cfa9ceaf89bfbc4745035479a7d8edee9b4465e64de"},
      {"hash":"e301017012201915dc0faad8e6783aca084a854c03553450efdabf977d57b4f22f73d5c53b50"},
      {"hash":"e301017012200b9fb71a129048c2a569433efc8e4d9155c54d598538be7f65ea26f665be1e84"},
      {"hash":"e30101701220d37944e3fb05213d45416fa634cf9e10ec1f43d3bf72c4eb3062ae6cc4ed9b08"},
      {"hash":"e3010170122059390dca66ba8713a9c323925bf768612f7dd16298c13a07a6b47cb5af4236e6"},
      {"hash":"e30101701220daaf88ace8a3356559be5d6912d5d442916e3cc92664954526c9815d693dc32b"},
      {"hash":"e301017012203ae30594fdf56d7bfd686cef1a45c201024e9c10a792722ef07ba968c83c064d"},
      {"hash":"e3010170122016e5eba0bbd32fc1ff17d80d1247fc67432705cd85731458b52febb84fdd6408"},
      {"hash":"e3010170122014fe2c2186cbf9d15ff61e04054fd6b0a5dbd7f365a1807f6f3d3d3e93e50875"},
      {"hash":"e30101701220f23a7dad3ea7ad3f3553a98fb305148d285e4ebf66b427d85a2340f66d51da94"},
      {"hash":"e3010170122047a637c6af02904a8ae702ec74b3df5fd8914df6fb11c99446a36d890beeb7ee"},
      {"hash":"e30101701220776f1ff89f6196ae68414545f6c6a5314c35eee7406cb8591d607a2b0533cc86"}
    ],
    "thumbnail":"e30101701220e9876531554a7cb4f20d7ebbf9daef2253e6734ad9c96ba288586a9b88bef491"
  }
}`)
)

func TestMessengerSyncSettings(t *testing.T) {
	suite.Run(t, new(MessengerSyncSettingsSuite))
}

type MessengerSyncSettingsSuite struct {
	suite.Suite
	alice  *Messenger
	alice2 *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger

	ignoreTests bool
}

func (s *MessengerSyncSettingsSuite) SetupSuite() {
	s.ignoreTests = true
}

func (s *MessengerSyncSettingsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.alice = s.newMessenger()
	_, err := s.alice.Start()
	s.Require().NoError(err)

	s.alice2, err = newMessengerWithKey(s.shh, s.alice.identity, s.logger, nil)
	s.Require().NoError(err)

	prepAliceMessengersForPairing(&s.Suite, s.alice, s.alice2)
}

func (s *MessengerSyncSettingsSuite) TearDownTest() {
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
		CurrentNetwork:            "mainnet_rpc",
		DappsAddress:              types.HexToAddress("0x1122334455667788990011223344556677889900"),
		InstallationID:            "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:                    "0x1122334455667788990011223344556677889900",
		Name:                      "Test",
		Networks:                  &networks,
		PhotoPath:                 "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreviewPrivacy:            false,
		PublicKey:                 "0x04112233445566778899001122334455667788990011223344556677889900112233445566778899001122334455667788990011223344556677889900",
		SigningPhrase:             "yurt joey vibe",
		SendPushNotifications:     true,
		ProfilePicturesShowTo:     1,
		ProfilePicturesVisibility: 1,
		DefaultSyncPeriod:         86400,
		UseMailservers:            true,
		LinkPreviewRequestEnabled: true,
		SendStatusUpdates:         true,
		WalletRootAddress:         types.HexToAddress("0x1122334455667788990011223344556677889900")}

	err = m.settings.CreateSettings(setting, config)
	s.Require().NoError(err)

	err = m.settings.SaveSettingField(settings.PreferredName, &pf)
	s.Require().NoError(err)

	err = m.settings.SaveSettingField(settings.Currency, "eth")
	s.Require().NoError(err)

	return m
}

func (s *MessengerSyncSettingsSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	options := []Option{
		WithCustomLogger(s.logger),
		WithDatabaseConfig(tmpFile.Name(), "", sqlite.ReducedKDFIterationsNumber),
		WithDatasync(),
	}
	return s.newMessengerWithOptions(shh, privateKey, options)
}

func (s *MessengerSyncSettingsSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func pairTwoDevices(s *suite.Suite, device1, device2 *Messenger) {
	// Send pairing data
	response, err := device1.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Len(response.Chats(), 1)
	s.False(response.Chats()[0].Active)

	i, ok := device1.allInstallations.Load(device1.installationID)
	s.Require().True(ok)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		device2,
		func(r *MessengerResponse) bool {
			for _, installation := range r.Installations {
				if installation.ID == device1.installationID {
					return installation.InstallationMetadata != nil &&
						i.InstallationMetadata.Name == installation.InstallationMetadata.Name &&
						i.InstallationMetadata.DeviceType == installation.InstallationMetadata.DeviceType
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

func prepAliceMessengersForPairing(s *suite.Suite, alice1, alice2 *Messenger) {
	// Set Alice's installation metadata
	aim := &multidevice.InstallationMetadata{
		Name:       "alice's-device",
		DeviceType: "alice's-device-type",
	}
	err := alice1.SetInstallationMetadata(alice1.installationID, aim)
	s.Require().NoError(err)

	// Set Alice 2's installation metadata
	a2im := &multidevice.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alice2.SetInstallationMetadata(alice2.installationID, a2im)
	s.Require().NoError(err)
}

func (s *MessengerSyncSettingsSuite) TestSyncSettings() {
	// Pair alice's two devices
	pairTwoDevices(&s.Suite, s.alice2, s.alice)
	pairTwoDevices(&s.Suite, s.alice, s.alice2)

	// Check alice 1 settings values
	as, err := s.alice.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Exactly(settings.ProfilePicturesShowToContactsOnly, as.ProfilePicturesShowTo)
	s.Require().Exactly(settings.ProfilePicturesVisibilityContactsOnly, as.ProfilePicturesVisibility)

	// Check alice 2 settings values
	aos, err := s.alice2.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Exactly(settings.ProfilePicturesShowToContactsOnly, aos.ProfilePicturesShowTo)
	s.Require().Exactly(settings.ProfilePicturesVisibilityContactsOnly, aos.ProfilePicturesVisibility)

	// Update alice ProfilePicturesVisibility setting
	err = s.alice.settings.SaveSettingField(settings.ProfilePicturesVisibility, settings.ProfilePicturesVisibilityEveryone)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		mr, err := s.alice2.RetrieveAll()
		if err != nil {
			return err
		}

		if len(mr.Settings) == 0 {
			return errors.New("sync settings not in MessengerResponse")
		}

		return nil
	})
	s.Require().NoError(err)

	// Check alice 2 settings values
	aos, err = s.alice2.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Equal(settings.ProfilePicturesVisibilityEveryone, aos.ProfilePicturesVisibility)

	// Alice 2 updated a setting which triggers the sync functionality
	err = s.alice2.settings.SaveSettingField(settings.ProfilePicturesShowTo, settings.ProfilePicturesShowToEveryone)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		mr, err := s.alice.RetrieveAll()
		if err != nil {
			return err
		}

		if len(mr.Settings) == 0 {
			return errors.New("sync settings not in MessengerResponse")
		}

		return nil
	})
	s.Require().NoError(err)

	// Check alice 1 settings values
	as, err = s.alice.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Exactly(settings.ProfilePicturesShowToEveryone, as.ProfilePicturesShowTo)
}

func (s *MessengerSyncSettingsSuite) TestSyncSettings_StickerPacks() {
	if s.ignoreTests {
		s.T().Skip("Currently sticker pack syncing has been deactivated, testing to resume after sticker packs works correctly")
		return
	}

	// Check alice 1 settings values
	as, err := s.alice.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Nil(as.StickerPacksInstalled)
	s.Require().Nil(as.StickerPacksPending)
	s.Require().Nil(as.StickersRecentStickers)

	// Check alice 2 settings values
	aos, err := s.alice2.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Nil(aos.StickerPacksInstalled)
	s.Require().Nil(aos.StickerPacksPending)
	s.Require().Nil(aos.StickersRecentStickers)

	// Pair devices. Allows alice to send to alicesOtherDevice
	pairTwoDevices(&s.Suite, s.alice2, s.alice)

	// Add sticker pack to alice device
	stickerPacks := make(stickers.StickerPackCollection)
	err = json.Unmarshal(rawSticker, &stickerPacks)
	s.Require().NoError(err)

	err = s.alice.settings.SaveSettingField(settings.StickersPacksInstalled, stickerPacks)
	s.Require().NoError(err)

	as, err = s.alice.settings.GetSettings()
	s.Require().NoError(err)
	spi, err := as.StickerPacksInstalled.MarshalJSON()
	s.Require().NoError(err)
	s.Require().Equal(2169, len(spi))

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		mr, err := s.alice2.RetrieveAll()
		if err != nil {
			return err
		}

		if len(mr.Settings) == 0 {
			return errors.New("sync settings not in MessengerResponse")
		}

		return nil
	})
	s.Require().NoError(err)

	aos, err = s.alice2.settings.GetSettings()
	s.Require().NoError(err)
	ospi, err := aos.StickerPacksInstalled.MarshalJSON()
	s.Require().NoError(err)
	s.Require().Exactly(spi, ospi)
}

func (s *MessengerSyncSettingsSuite) TestSyncSettings_PreferredName() {
	if s.ignoreTests {
		s.T().Skip("Currently preferred syncing has been deactivated, testing to resume after ens names also sync")
		return
	}

	// Check alice 1 settings values
	as, err := s.alice.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Equal(pf, *as.PreferredName)

	// Check alice 2 settings values
	aos, err := s.alice2.settings.GetSettings()
	s.Require().NoError(err)
	s.Require().Nil(aos.PreferredName)

	// Pair devices. Allows alice to send to alicesOtherDevice
	pairTwoDevices(&s.Suite, s.alice2, s.alice)

	// Update Alice's PreferredName
	err = s.alice.settings.SaveSettingField(settings.PreferredName, pf2)
	s.Require().NoError(err)

	apn, err := s.alice.settings.GetPreferredUsername()
	s.Require().NoError(err)
	s.Require().Equal(pf2, apn)

	// Wait for the sync message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		mr, err := s.alice2.RetrieveAll()
		if err != nil {
			return err
		}

		if len(mr.Settings) == 0 {
			return errors.New("sync settings not in MessengerResponse")
		}

		return nil
	})
	s.Require().NoError(err)

	opn, err := s.alice2.settings.GetPreferredUsername()
	s.Require().NoError(err)
	s.Require().Equal(pf2, opn)
}
