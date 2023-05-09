package protocol

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/keycards"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerSyncKeycardsStateSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncKeycardsStateSuite))
}

type MessengerSyncKeycardsStateSuite struct {
	suite.Suite
	main       *Messenger // main instance of Messenger paired with `other`
	other      *Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerSyncKeycardsStateSuite) SetupTest() {
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

func (s *MessengerSyncKeycardsStateSuite) TearDownTest() {
	s.Require().NoError(s.other.Shutdown())
	s.Require().NoError(s.main.Shutdown())
}

func (s *MessengerSyncKeycardsStateSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)

	return messenger
}

func sameKeycards(a, b *keycards.Keycard) bool {
	same := a.KeycardUID == b.KeycardUID &&
		a.KeyUID == b.KeyUID &&
		a.KeycardName == b.KeycardName &&
		a.KeycardLocked == b.KeycardLocked &&
		a.LastUpdateClock == b.LastUpdateClock &&
		len(a.AccountsAddresses) == len(b.AccountsAddresses)

	if same {
		for i := range a.AccountsAddresses {
			found := false
			for j := range b.AccountsAddresses {
				if a.AccountsAddresses[i] == b.AccountsAddresses[j] {
					found = true
					break
				}
			}

			if !found {
				return false
			}
		}
	}

	return same
}

func getKeycardsForTest() []*keycards.Keycard {
	keycard1 := keycards.Keycard{
		KeycardUID:        "00000000000000000000000000000001",
		KeycardName:       "Card01",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}, {0x03}, {0x04}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000001",
		LastUpdateClock:   100,
	}
	keycard2 := keycards.Keycard{
		KeycardUID:        "00000000000000000000000000000002",
		KeycardName:       "Card02",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000002",
		LastUpdateClock:   200,
	}
	keycard3 := keycards.Keycard{
		KeycardUID:        "00000000000000000000000000000003",
		KeycardName:       "Card02 Copy",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000002",
		LastUpdateClock:   300,
	}
	keycard4 := keycards.Keycard{
		KeycardUID:        "00000000000000000000000000000004",
		KeycardName:       "Card04",
		KeycardLocked:     false,
		AccountsAddresses: []types.Address{{0x01}, {0x02}, {0x03}},
		KeyUID:            "0000000000000000000000000000000000000000000000000000000000000004",
		LastUpdateClock:   400,
	}

	return []*keycards.Keycard{&keycard1, &keycard2, &keycard3, &keycard4}
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverHasNoKeycards() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	allKeycardsToSync := getKeycardsForTest()
	for _, kp := range allKeycardsToSync {
		addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Trigger's a sync between devices
	err := s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.AllKnownKeycards()) == len(allKeycardsToSync)
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(len(allKeycardsToSync), len(syncedKeycards))
	s.Require().True(haveSameElements(syncedKeycards, allKeycardsToSync, sameKeycards))
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverHasKeycardsOlderThanSender() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	allKeycardsToSync := getKeycardsForTest()
	for _, kp := range allKeycardsToSync {
		addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Add keycards on receiver
	keycardsOnReceiver := getKeycardsForTest()[:2]
	keycardsOnReceiver[0].KeycardName = "CardNameToBeChanged-0"
	keycardsOnReceiver[0].AccountsAddresses = keycardsOnReceiver[0].AccountsAddresses[2:3]
	keycardsOnReceiver[0].LastUpdateClock = keycardsOnReceiver[0].LastUpdateClock - 1
	keycardsOnReceiver[1].KeycardName = "CardNameToBeChanged-1"
	keycardsOnReceiver[1].LastUpdateClock = keycardsOnReceiver[1].LastUpdateClock - 1

	for _, kp := range keycardsOnReceiver {
		addedKc, addedAccs, err := dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Trigger's a sync between devices
	err := s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.AllKnownKeycards()) == len(allKeycardsToSync)
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(len(allKeycardsToSync), len(syncedKeycards))
	s.Require().True(haveSameElements(syncedKeycards, allKeycardsToSync, sameKeycards))
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfKeycardsWereDeletedOnSenderSide() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	allKeycardsToSync := getKeycardsForTest()[:2]
	for _, kp := range allKeycardsToSync {
		addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Add keycards on receiver
	keycardsOnReceiver := getKeycardsForTest()
	for _, kp := range keycardsOnReceiver {
		addedKc, addedAccs, err := dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Trigger's a sync between devices
	err := s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.AllKnownKeycards()) == len(allKeycardsToSync)
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(len(allKeycardsToSync), len(syncedKeycards))
	s.Require().True(haveSameElements(syncedKeycards, allKeycardsToSync, sameKeycards))
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverHasNewerKeycardsThanTheSameAreDeletedOnSenderSide() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	allKeycardsToSync := getKeycardsForTest()[:2]
	for _, kp := range allKeycardsToSync {
		addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Add keycards on receiver
	keycardsOnReceiver := getKeycardsForTest()
	clock, _ := s.other.getLastClockWithRelatedChat()
	keycardsOnReceiver[2].KeycardName = "NewerCardName-2"
	keycardsOnReceiver[2].LastUpdateClock = clock + 1000
	keycardsOnReceiver[3].KeycardName = "NewerCardName-3"
	keycardsOnReceiver[3].LastUpdateClock = clock + 1000
	for _, kp := range keycardsOnReceiver {
		addedKc, addedAccs, err := dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Trigger's a sync between devices
	err := s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.AllKnownKeycards()) >= len(allKeycardsToSync)
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(len(keycardsOnReceiver), len(syncedKeycards))
	for _, kc := range allKeycardsToSync {
		s.Require().True(contains(syncedKeycards, kc, sameKeycards))
	}
	for _, kc := range keycardsOnReceiver {
		s.Require().True(contains(syncedKeycards, kc, sameKeycards))
	}
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverAndSenderHasNoKeycardsInCommon() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	allKeycardsToSync := getKeycardsForTest()[:2]
	for _, kp := range allKeycardsToSync {
		addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Add keycards on receiver
	keycardsOnReceiver := getKeycardsForTest()[2:]
	clock, _ := s.other.getLastClockWithRelatedChat()
	keycardsOnReceiver[0].KeycardName = "NewerCardName-0"
	keycardsOnReceiver[0].LastUpdateClock = clock + 1000
	keycardsOnReceiver[1].KeycardName = "NewerCardName-1"
	keycardsOnReceiver[1].LastUpdateClock = clock + 1000

	for _, kp := range keycardsOnReceiver {
		addedKc, addedAccs, err := dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Trigger's a sync between devices
	err := s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.AllKnownKeycards()) >= len(allKeycardsToSync)
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(len(allKeycardsToSync)+len(keycardsOnReceiver), len(syncedKeycards))
	for _, kc := range allKeycardsToSync {
		s.Require().True(contains(syncedKeycards, kc, sameKeycards))
	}
	for _, kc := range keycardsOnReceiver {
		s.Require().True(contains(syncedKeycards, kc, sameKeycards))
	}
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverHasNewerKeycardThanSender() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	allKeycardsToSync := getKeycardsForTest()
	for _, kp := range allKeycardsToSync {
		addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Add keycards on receiver
	keycardsOnReceiver := getKeycardsForTest()[2:]
	clock, _ := s.other.getLastClockWithRelatedChat()
	keycardsOnReceiver[0].KeycardName = "NewerCardName-0"
	keycardsOnReceiver[0].LastUpdateClock = clock + 1000
	keycardsOnReceiver[1].KeycardName = "NewerCardName-1"
	keycardsOnReceiver[1].LastUpdateClock = clock + 1000

	for _, kp := range keycardsOnReceiver {
		addedKc, addedAccs, err := dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*kp)
		s.Require().NoError(err)
		s.Require().Equal(true, addedKc)
		s.Require().Equal(false, addedAccs)
	}

	// Trigger's a sync between devices
	err := s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.AllKnownKeycards()) == len(allKeycardsToSync)
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(len(allKeycardsToSync), len(syncedKeycards))
	for _, kc := range allKeycardsToSync[:2] {
		s.Require().True(contains(syncedKeycards, kc, sameKeycards))
	}
	for _, kc := range keycardsOnReceiver {
		s.Require().True(contains(syncedKeycards, kc, sameKeycards))
	}
}
