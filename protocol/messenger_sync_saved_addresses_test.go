package protocol

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
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

	s.main = s.newMessenger(s.shh)
	s.privateKey = s.main.identity
	// Start the main messenger in order to receive installations
	_, err := s.main.Start()
	s.Require().NoError(err)

	// Create new device and add main account to
	s.other, err = newMessengerWithKey(s.shh, s.main.identity, s.logger, nil)
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
	s.Require().NoError(s.main.Shutdown())
}

func (s *MessengerSyncSavedAddressesSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
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
	for _, v := range a {
		if !contains(b, v, isEqual) {
			return false
		}
	}
	return true
}

func savedAddressDataIsEqual(a, b wallet.SavedAddress) bool {
	return a.Address == b.Address && a.ChainID == b.ChainID && a.Name == b.Name && a.Favourite == b.Favourite
}

func (s *MessengerSyncSavedAddressesSuite) TestSyncExistingSavedAddresses() {
	var testChainID1 uint64 = 1
	var testChainID2 uint64 = 2
	var testAddress1 = common.Address{1}

	// Add saved addresses to main device
	sa1 := wallet.SavedAddress{
		Address:   testAddress1,
		Name:      "TestC1A1",
		Favourite: false,
		ChainID:   testChainID1,
	}
	sa2 := wallet.SavedAddress{
		Address:   testAddress1,
		Name:      "TestC2A1",
		Favourite: true,
		ChainID:   testChainID2,
	}

	savedAddressesManager := wallet.NewSavedAddressesManager(s.main.persistence.db)

	_, err := savedAddressesManager.UpdateMetadataAndUpsertSavedAddress(sa1)
	s.Require().NoError(err)
	_, err = savedAddressesManager.UpdateMetadataAndUpsertSavedAddress(sa2)
	s.Require().NoError(err)

	// Trigger's a sync between devices
	err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait and check that saved addresses are synced
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			fmt.Println("LENG", len(r.SavedAddresses()))
			if len(r.SavedAddresses()) == 2 {
				sas := r.SavedAddresses()
				s.Require().True(haveSameElements([]wallet.SavedAddress{sa1, sa2}, []wallet.SavedAddress{*sas[0], *sas[1]}, savedAddressDataIsEqual))
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
	s.Require().True(haveSameElements([]wallet.SavedAddress{sa1, sa2}, savedAddresses, savedAddressDataIsEqual))
}

func (s *MessengerSyncSavedAddressesSuite) TestSyncSavedAddresses() {
	var testChainID1 uint64 = 1
	var testAddress1 = common.Address{1}
	var testAddress2 = common.Address{2}

	// Add saved addresses to main device
	sa1 := wallet.SavedAddress{
		Address:   testAddress1,
		Name:      "TestC1A1",
		Favourite: false,
		ChainID:   testChainID1,
	}
	sa2 := wallet.SavedAddress{
		Address:   testAddress2,
		Name:      "TestC1A2",
		Favourite: true,
		ChainID:   testChainID1,
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
				s.Require().True(haveSameElements([]wallet.SavedAddress{sa1, sa2}, []wallet.SavedAddress{*sas[0], *sas[1]}, savedAddressDataIsEqual))
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
	s.Require().True(haveSameElements([]wallet.SavedAddress{sa1, sa2}, savedAddresses, savedAddressDataIsEqual))
}

func (s *MessengerSyncSavedAddressesSuite) TestSyncDeletesOfSavedAddresses() {
	var testChainID1 uint64 = 1
	var testAddress1 = common.Address{1}
	var testAddress2 = common.Address{2}

	// Add saved addresses to main device
	sa1 := wallet.SavedAddress{
		Address:   testAddress1,
		Name:      "TestC1A1",
		Favourite: false,
		ChainID:   testChainID1,
	}
	sa2 := wallet.SavedAddress{
		Address:   testAddress2,
		Name:      "TestC1A2",
		Favourite: true,
		ChainID:   testChainID1,
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
				s.Require().True(haveSameElements([]wallet.SavedAddress{sa1, sa2}, []wallet.SavedAddress{*sas[0], *sas[1]}, savedAddressDataIsEqual))
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

	// Delete saved addresses from the other device
	err = s.main.DeleteSavedAddress(context.Background(), sa1.ChainID, sa1.Address)
	s.Require().NoError(err)

	// Wait and check that saved addresses are synced
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			if len(r.SavedAddresses()) == 1 {
				// We expect the deleted event to report only address and chain ID
				s.Require().Equal(sa1.Address, r.SavedAddresses()[0].Address)
				s.Require().Equal(sa1.ChainID, r.SavedAddresses()[0].ChainID)
				s.Require().Equal("", r.SavedAddresses()[0].Name)
				s.Require().Equal(false, r.SavedAddresses()[0].Favourite)
				return true
			}
			return false
		},
		"expected to receive one change",
	)
	s.Require().NoError(err)

	savedAddresses, err = s.other.savedAddressesManager.GetSavedAddresses()
	s.Require().NoError(err)
	s.Require().Equal(1, len(savedAddresses))
	s.Require().True(haveSameElements([]wallet.SavedAddress{sa2}, savedAddresses, savedAddressDataIsEqual))
}
