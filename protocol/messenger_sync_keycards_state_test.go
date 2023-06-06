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
	"github.com/status-im/status-go/multiaccounts/accounts"
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

	// Pre-condition - both sides have to know about keypairs migrated to a keycards
	kp1 := accounts.GetProfileKeypairForTest(true, true, true)
	kp2 := accounts.GetSeedImportedKeypair1ForTest()
	kp3 := accounts.GetSeedImportedKeypair2ForTest()

	err = s.main.settings.SaveOrUpdateKeypair(kp1)
	s.Require().NoError(err)
	err = s.main.settings.SaveOrUpdateKeypair(kp2)
	s.Require().NoError(err)
	err = s.main.settings.SaveOrUpdateKeypair(kp3)
	s.Require().NoError(err)
	dbKeypairs, err := s.main.settings.GetKeypairs()
	s.Require().NoError(err)
	s.Require().Equal(3, len(dbKeypairs))

	err = s.other.SaveOrUpdateKeypair(kp1)
	s.Require().NoError(err)
	err = s.other.SaveOrUpdateKeypair(kp2)
	s.Require().NoError(err)
	err = s.other.SaveOrUpdateKeypair(kp3)
	s.Require().NoError(err)
	dbKeypairs, err = s.other.settings.GetKeypairs()
	s.Require().NoError(err)
	s.Require().Equal(3, len(dbKeypairs))
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

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverHasNoKeycards() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	keycard2Copy := accounts.GetKeycardForSeedImportedKeypair1ForTest()
	keycard2Copy.KeycardUID = keycard2Copy.KeycardUID + "C"
	keycard2Copy.KeycardName = keycard2Copy.KeycardName + "Copy"
	keycard2Copy.LastUpdateClock = keycard2Copy.LastUpdateClock + 1

	keycard3 := accounts.GetKeycardForSeedImportedKeypair2ForTest()

	// Add keycards on sender
	addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard1)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2Copy)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard3)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)

	// Trigger's a sync between devices
	err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 3 && len(r.Keycards) == 4
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(4, len(syncedKeycards))
	s.Require().True(contains(syncedKeycards, keycard1, accounts.SameKeycards))
	s.Require().True(contains(syncedKeycards, keycard2, accounts.SameKeycards))
	s.Require().True(contains(syncedKeycards, keycard2Copy, accounts.SameKeycards))
	s.Require().True(contains(syncedKeycards, keycard3, accounts.SameKeycards))
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverHasKeycardsOlderThanSender() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	// Add keycards on sender
	addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard1)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)

	// Add keycards on receiver - partially
	keycardR1 := accounts.GetProfileKeycardForTest()
	keycardR1.KeycardName = "CardNameToBeChanged-0"
	keycardR1.AccountsAddresses = keycardR1.AccountsAddresses[2:3]
	keycardR1.LastUpdateClock = keycardR1.LastUpdateClock - 1

	keycardR2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()
	keycardR2.KeycardName = "CardNameToBeChanged-1"
	keycardR2.LastUpdateClock = keycardR2.LastUpdateClock - 1

	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycardR1)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycardR2)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)

	// Trigger's a sync between devices
	err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 3 && len(r.Keycards) == 2
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(syncedKeycards))
	s.Require().True(contains(syncedKeycards, keycard1, accounts.SameKeycards))
	s.Require().True(contains(syncedKeycards, keycard2, accounts.SameKeycards))
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverHasKeycardsNewerThanSender() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	// Add keycards on sender
	addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard1)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)

	// Add keycards on receiver - partially
	keycardR1 := accounts.GetProfileKeycardForTest()
	keycardR1.KeycardName = "CardNameToBeChanged-0"
	keycardR1.AccountsAddresses = keycardR1.AccountsAddresses[2:3]
	keycardR1.LastUpdateClock = keycardR1.LastUpdateClock + 1

	keycardR2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()
	keycardR2.KeycardName = "CardNameToBeChanged-1"
	keycardR2.LastUpdateClock = keycardR2.LastUpdateClock + 1

	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycardR1)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycardR2)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)

	// Trigger's a sync between devices
	err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 3 && len(r.Keycards) == 2
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(syncedKeycards))
	s.Require().True(contains(syncedKeycards, keycardR1, accounts.SameKeycards))
	s.Require().True(contains(syncedKeycards, keycardR2, accounts.SameKeycards))
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfKeycardsWereDeletedOnSenderSide() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	keycard2Copy := accounts.GetKeycardForSeedImportedKeypair1ForTest()
	keycard2Copy.KeycardUID = keycard2Copy.KeycardUID + "C"
	keycard2Copy.KeycardName = keycard2Copy.KeycardName + "Copy"
	keycard2Copy.LastUpdateClock = keycard2Copy.LastUpdateClock + 1

	keycard3 := accounts.GetKeycardForSeedImportedKeypair2ForTest()

	// Add keycards on sender
	addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard1)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)

	// Add keycards on receiver
	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard1)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2Copy)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard3)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)

	// Trigger's a sync between devices
	err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 3 && len(r.Keycards) == 2
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(syncedKeycards))
	s.Require().True(contains(syncedKeycards, keycard1, accounts.SameKeycards))
	s.Require().True(contains(syncedKeycards, keycard2, accounts.SameKeycards))
}

func (s *MessengerSyncKeycardsStateSuite) TestSyncKeycardsIfReceiverAndSenderHasNoKeycardsInCommon() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	// Add keycards on sender
	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	keycard2Copy := accounts.GetKeycardForSeedImportedKeypair1ForTest()
	keycard2Copy.KeycardUID = keycard2Copy.KeycardUID + "C"
	keycard2Copy.KeycardName = keycard2Copy.KeycardName + "Copy"
	keycard2Copy.LastUpdateClock = keycard2Copy.LastUpdateClock + 1

	keycard3 := accounts.GetKeycardForSeedImportedKeypair2ForTest()

	// Add keycards on sender
	addedKc, addedAccs, err := senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = senderDb.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard2Copy)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)

	// Add keycards on receiver
	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard1)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)
	addedKc, addedAccs, err = dbOnReceiver.AddKeycardOrAddAccountsIfKeycardIsAdded(*keycard3)
	s.Require().NoError(err)
	s.Require().Equal(true, addedKc)
	s.Require().Equal(false, addedAccs)

	// Trigger's a sync between devices
	err = s.main.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 3 && len(r.Keycards) == 2
		},
		"expected to receive keycards",
	)
	s.Require().NoError(err)

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(syncedKeycards))
	s.Require().True(contains(syncedKeycards, keycard2, accounts.SameKeycards))
	s.Require().True(contains(syncedKeycards, keycard2Copy, accounts.SameKeycards))
}
