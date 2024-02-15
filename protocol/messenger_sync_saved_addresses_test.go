package protocol

import (
	"context"
	"crypto/ecdsa"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/waku"
)

func TestMessengerSyncSavedAddressesSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncSavedAddressesSuite))
}

type MessengerSyncSavedAddressesSuite struct {
	suite.Suite
	main       *Messenger // main instance of Messenger paired with `other`
	other      *Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerSyncSavedAddressesSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.main = s.newMessenger(s.logger.Named("main"))
	s.privateKey = s.main.identity
	// Start the main messenger in order to receive installations
	_, err := s.main.Start()
	s.Require().NoError(err)

	// Create new device and add main account to
	s.other, err = newMessengerWithKey(s.shh, s.main.identity, s.logger.Named("other"), nil)
	s.Require().NoError(err)

	// Pair devices (main and other)
	imOther := &multidevice.InstallationMetadata{
		Name:       "other-device",
		DeviceType: "other-device-type",
	}
	err = s.other.SetInstallationMetadata(s.other.installationID, imOther)
	s.Require().NoError(err)
	response, err := s.other.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		s.main,
		func(r *MessengerResponse) bool { return len(r.Installations) > 0 },
		"installation not received",
	)
	s.Require().NoError(err)

	err = s.main.EnableInstallation(s.other.installationID)
	s.Require().NoError(err)
}

func (s *MessengerSyncSavedAddressesSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.main)
}

func (s *MessengerSyncSavedAddressesSuite) newMessenger(logger *zap.Logger) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, logger, nil)
	s.Require().NoError(err)

	return messenger
}

// Helpers duplicate of wallet test. Could not import it from saved_addresses_test.go

func contains[T comparable](container []T, element T, isEqual func(T, T) bool) bool {
	for _, e := range container {
		if isEqual(e, element) {
			return true
		}
	}
	return false
}

func haveSameElements[T comparable](a []T, b []T, isEqual func(T, T) bool) bool {
	if len(a) != len(b) {
		return false
	}
	for _, v := range a {
		if !contains(b, v, isEqual) {
			return false
		}
	}
	return true
}

func savedAddressDataIsEqual(a, b *wallet.SavedAddress) bool {
	return a.Address == b.Address && a.IsTest == b.IsTest && a.Name == b.Name &&
		a.ENSName == b.ENSName && a.ChainShortNames == b.ChainShortNames && a.ColorID == b.ColorID
}

func (s *MessengerSyncSavedAddressesSuite) TestSyncExistingSavedAddresses() {
	var isTestChain1 bool = false
	var isTestChain2 bool = true
	var testAddress1 = common.Address{1}

	// Add saved addresses to main device
	sa1 := wallet.SavedAddress{
		Address: testAddress1,
		Name:    "TestC1A1",
		IsTest:  isTestChain1,
	}
	sa2 := wallet.SavedAddress{
		ENSName: "test.ens.eth",
		Name:    "TestC2A1",
		IsTest:  isTestChain2,
	}

	err := s.main.UpsertSavedAddress(context.Background(), sa1)
	s.Require().NoError(err)
	err = s.main.UpsertSavedAddress(context.Background(), sa2)
	s.Require().NoError(err)

	//// Trigger's a sync between devices
	//err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	//s.Require().NoError(err)

	// Wait and check that saved addresses are synced
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			if len(r.SavedAddresses()) == 2 {
				sas := r.SavedAddresses()
				s.Require().True(haveSameElements([]*wallet.SavedAddress{&sa1, &sa2}, []*wallet.SavedAddress{sas[0], sas[1]}, savedAddressDataIsEqual))
				return true
			}
			return false
		},
		"expected to receive two changes",
	)
	s.Require().NoError(err)

	savedAddresses, err := s.other.savedAddressesManager.GetSavedAddresses()
	s.Require().NoError(err)
	s.Require().Equal(2, len(savedAddresses))
	s.Require().True(haveSameElements([]*wallet.SavedAddress{&sa1, &sa2}, savedAddresses, savedAddressDataIsEqual))
}

func (s *MessengerSyncSavedAddressesSuite) TestSyncSavedAddresses() {
	var isTestChain1 bool = true
	var testAddress1 = common.Address{1}

	// Add saved addresses to main device
	sa1 := wallet.SavedAddress{
		Address: testAddress1,
		Name:    "TestC1A1",
		IsTest:  isTestChain1,
	}
	sa2 := wallet.SavedAddress{
		ENSName: "test.ens.eth",
		Name:    "TestC1A2",
		IsTest:  isTestChain1,
	}

	err := s.main.UpsertSavedAddress(context.Background(), sa1)
	s.Require().NoError(err)
	err = s.main.UpsertSavedAddress(context.Background(), sa2)
	s.Require().NoError(err)

	// Wait and check that saved addresses are synced
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			if len(r.SavedAddresses()) == 2 {
				sas := r.SavedAddresses()
				s.Require().True(haveSameElements([]*wallet.SavedAddress{&sa1, &sa2}, []*wallet.SavedAddress{sas[0], sas[1]}, savedAddressDataIsEqual))
				return true
			}
			return false
		},
		"expected to receive two changes",
	)
	s.Require().NoError(err)

	savedAddresses, err := s.other.savedAddressesManager.GetSavedAddresses()
	s.Require().NoError(err)
	s.Require().Equal(2, len(savedAddresses))
	s.Require().True(haveSameElements([]*wallet.SavedAddress{&sa1, &sa2}, savedAddresses, savedAddressDataIsEqual))
}

func (s *MessengerSyncSavedAddressesSuite) requireSavedAddressesEqual(a, b []*wallet.SavedAddress) {
	sort.Slice(a, func(i, j int) bool {
		return a[i].Address.Hex() < a[j].Address.Hex()
	})
	sort.Slice(b, func(i, j int) bool {
		return b[i].Address.Hex() < b[j].Address.Hex()
	})
	s.Require().True(reflect.DeepEqual(a, b))
}

func (s *MessengerSyncSavedAddressesSuite) testSyncDeletesOfSavedAddresses(testModeMain bool, testModeOther bool) {

	sa1 := &wallet.SavedAddress{
		Address: common.Address{1},
		Name:    "TestC1A1",
		IsTest:  true,
	}
	sa2 := &wallet.SavedAddress{
		Address: common.Address{2},
		Name:    "TestC1A2",
		IsTest:  false,
	}

	err := s.main.settings.SaveSettingField(settings.TestNetworksEnabled, testModeMain)
	s.Require().NoError(err)
	err = s.other.settings.SaveSettingField(settings.TestNetworksEnabled, testModeOther)
	s.Require().NoError(err)

	// Add saved addresses to main device
	err = s.main.UpsertSavedAddress(context.Background(), *sa1)
	s.Require().NoError(err)
	err = s.main.UpsertSavedAddress(context.Background(), *sa2)
	s.Require().NoError(err)

	// Wait and check that saved addresses are synced
	{
		response, err := WaitOnMessengerResponse(
			s.other,
			func(r *MessengerResponse) bool {
				return len(r.SavedAddresses()) == 2
			},
			"expected to receive two changes",
		)
		s.Require().NoError(err)

		otherSavedAddresses := response.SavedAddresses()
		s.Require().Len(otherSavedAddresses, 2)

		// Check that the UpdateClock was bumped
		s.Require().GreaterOrEqual(otherSavedAddresses[0].CreatedAt, int64(0))
		s.Require().GreaterOrEqual(otherSavedAddresses[1].CreatedAt, int64(0))
		s.Require().Greater(otherSavedAddresses[0].UpdateClock, uint64(0))
		s.Require().Greater(otherSavedAddresses[1].UpdateClock, uint64(0))

		// Reset the UpdateClock to 0 for comparison
		otherSavedAddresses[0].CreatedAt = 0
		otherSavedAddresses[1].CreatedAt = 0
		otherSavedAddresses[0].UpdateClock = 0
		otherSavedAddresses[1].UpdateClock = 0
		s.requireSavedAddressesEqual([]*wallet.SavedAddress{sa1, sa2}, otherSavedAddresses)

		// Ensure the messenger actually has the saved addresses, not just the response
		savedAddresses, err := s.other.savedAddressesManager.GetSavedAddresses()
		s.Require().NoError(err)
		s.Require().Len(savedAddresses, 2)

		// Reset the UpdateClock to 0 for comparison
		savedAddresses[0].CreatedAt = 0
		savedAddresses[1].CreatedAt = 0
		savedAddresses[0].UpdateClock = 0
		savedAddresses[1].UpdateClock = 0
		s.requireSavedAddressesEqual([]*wallet.SavedAddress{sa1, sa2}, savedAddresses)
	}

	// Delete saved address 1 (test mode = true) and sync with the other device
	{
		err = s.main.DeleteSavedAddress(context.Background(), sa1.Address, sa1.IsTest)
		s.Require().NoError(err)

		// Ensure the removal
		savedAddresses, err := s.main.savedAddressesManager.GetSavedAddresses()
		s.Require().NoError(err)
		s.Require().Len(savedAddresses, 1)
		sa2.CreatedAt = savedAddresses[0].CreatedAt     // force same value
		sa2.UpdateClock = savedAddresses[0].UpdateClock // force same value
		s.Require().Equal(sa2, savedAddresses[0])

		// Wait other device to receive the change
		response, err := WaitOnMessengerResponse(
			s.other,
			func(r *MessengerResponse) bool {
				return len(r.SavedAddresses()) == 1
			},
			"saved address removal wasn't received",
		)
		s.Require().NoError(err)

		// We expect the delete event to report address, ens, isTest
		sa := response.SavedAddresses()[0]
		s.Require().Equal(sa1.Address, sa.Address)
		s.Require().Equal(sa1.IsTest, sa.IsTest)
		s.Require().Equal("", sa.Name)

		// Ensure the messenger doesn't return the removed address
		savedAddresses, err = s.other.savedAddressesManager.GetSavedAddresses()
		s.Require().NoError(err)
		s.Require().Len(savedAddresses, 1)
		savedAddresses[0].CreatedAt = sa2.CreatedAt // force same value
		s.Require().Equal(sa2, savedAddresses[0])
	}

	// Delete saved address 2 (test mode = false) and sync with the other device
	{
		err = s.main.DeleteSavedAddress(context.Background(), sa2.Address, sa2.IsTest)
		s.Require().NoError(err)

		// Ensure the removal
		savedAddresses, err := s.main.savedAddressesManager.GetSavedAddresses()
		s.Require().NoError(err)
		s.Require().Len(savedAddresses, 0)

		// Wait other device to receive the change
		response, err := WaitOnMessengerResponse(
			s.other,
			func(r *MessengerResponse) bool {
				return len(r.SavedAddresses()) == 1
			},
			"expected to receive one change",
		)
		s.Require().NoError(err)

		sa := response.SavedAddresses()[0]
		// We expect the deleted event to report address, ens, isTest
		s.Require().Equal(sa2.Address, sa.Address)
		s.Require().Equal(sa2.IsTest, sa.IsTest)
		s.Require().Equal("", sa.Name)

		savedAddresses, err = s.other.savedAddressesManager.GetSavedAddresses()
		s.Require().NoError(err)
		s.Require().Len(savedAddresses, 0)
	}
}

func (s *MessengerSyncSavedAddressesSuite) TestSyncDeletesOfSavedAddresses() {
	testCases := []struct {
		Name          string
		TestModeMain  bool
		TestModeOther bool
	}{
		{
			Name:          "same test mode on both devices",
			TestModeMain:  true,
			TestModeOther: true,
		},
		{
			Name:          "different test mode on devices",
			TestModeMain:  true,
			TestModeOther: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			s.testSyncDeletesOfSavedAddresses(tc.TestModeMain, tc.TestModeOther)
		})
	}
}
