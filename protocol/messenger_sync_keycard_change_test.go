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

func TestMessengerSyncKeycardChangeSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncKeycardChangeSuite))
}

type MessengerSyncKeycardChangeSuite struct {
	suite.Suite
	main       *Messenger // main instance of Messenger paired with `other`
	other      *Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerSyncKeycardChangeSuite) SetupTest() {
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
	// kp3 := accounts.GetSeedImportedKeypair2ForTest()
	kp1.Clock = 1
	kp2.Clock = 1
	// kp3.Clock = 1

	err = s.main.settings.SaveOrUpdateKeypair(kp1)
	s.Require().NoError(err)
	err = s.main.settings.SaveOrUpdateKeypair(kp2)
	s.Require().NoError(err)
	// err = s.main.settings.SaveOrUpdateKeypair(kp3)
	// s.Require().NoError(err)
	dbKeypairs, err := s.main.settings.GetActiveKeypairs()
	s.Require().NoError(err)
	s.Require().Equal(2, len(dbKeypairs))

	kp1.Clock = 0
	kp2.Clock = 0
	// kp3.Clock = 0

	err = s.other.settings.SaveOrUpdateKeypair(kp1)
	s.Require().NoError(err)
	err = s.other.settings.SaveOrUpdateKeypair(kp2)
	s.Require().NoError(err)
	// err = s.other.settings.SaveOrUpdateKeypair(kp3)
	// s.Require().NoError(err)
	dbKeypairs, err = s.other.settings.GetActiveKeypairs()
	s.Require().NoError(err)
	s.Require().Equal(2, len(dbKeypairs))
}

func (s *MessengerSyncKeycardChangeSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.other)
	TearDownMessenger(&s.Suite, s.main)
}

func (s *MessengerSyncKeycardChangeSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)

	return messenger
}

func (s *MessengerSyncKeycardChangeSuite) TestAddingNewKeycards() {
	dbOnReceiver := s.other.settings

	// Add key cards on sender
	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	err := s.main.SaveOrUpdateKeycard(context.Background(), keycard1)
	s.Require().NoError(err)
	err = s.main.SaveOrUpdateKeycard(context.Background(), keycard2)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 2
		},
		"expected to receive keycard activities",
	)
	s.Require().NoError(err)

	senderKeycards, err := s.main.settings.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(senderKeycards))
	s.Require().True(accounts.Contains(senderKeycards, keycard1, accounts.SameKeycards))
	s.Require().True(accounts.Contains(senderKeycards, keycard2, accounts.SameKeycards))

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(syncedKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard1, accounts.SameKeycards))
	s.Require().True(accounts.Contains(syncedKeycards, keycard2, accounts.SameKeycards))
}

func (s *MessengerSyncKeycardChangeSuite) TestAddingAccountsToKeycard() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	keycard2 := accounts.GetKeycardForSeedImportedKeypair1ForTest()

	// Add keycard on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Add the same keycard on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Add additional accounts to sender
	err = s.main.SaveOrUpdateKeycard(context.Background(), keycard2)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 1
		},
		"expected to receive keycard activities",
	)
	s.Require().NoError(err)

	senderKeycards, err := senderDb.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(senderKeycards))
	s.Require().True(contains(senderKeycards, keycard1, accounts.SameKeycards))
	s.Require().True(contains(senderKeycards, keycard2, accounts.SameKeycards))

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(2, len(syncedKeycards))
	s.Require().True(contains(syncedKeycards, keycard1, accounts.SameKeycards))
	s.Require().True(contains(syncedKeycards, keycard2, accounts.SameKeycards))
}

func (s *MessengerSyncKeycardChangeSuite) TestRemovingAccountsFromKeycard() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	// Add keycard on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Add the same keycard on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Prepare expected keycard for comparison
	updatedKeycard1 := accounts.GetProfileKeycardForTest()
	updatedKeycard1.AccountsAddresses = updatedKeycard1.AccountsAddresses[2:]

	// Remove accounts from sender
	err = s.main.DeleteKeycardAccounts(context.Background(), keycard1.KeycardUID, keycard1.AccountsAddresses[:2])
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 1
		},
		"expected to receive keycard activities",
	)
	s.Require().NoError(err)

	senderKeycards, err := senderDb.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(1, len(senderKeycards))
	s.Require().True(contains(senderKeycards, updatedKeycard1, accounts.SameKeycards))

	syncedKeycards, err := dbOnReceiver.GetAllKnownKeycards()
	s.Require().NoError(err)
	s.Require().Equal(1, len(syncedKeycards))
	s.Require().True(contains(syncedKeycards, updatedKeycard1, accounts.SameKeycards))
}

func (s *MessengerSyncKeycardChangeSuite) TestRemovingAllAccountsForKeyUID() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	// Add keycard on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Add the same keycard on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Remove all accounts from sender for KeyUID
	err = s.main.DeleteAllKeycardsWithKeyUID(context.Background(), keycard1.KeyUID)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 1
		},
		"expected to receive keycard activities",
	)
	s.Require().NoError(err)

	senderKeycards, err := senderDb.GetKeycardsWithSameKeyUID(keycard1.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(0, len(senderKeycards))

	syncedKeycards, err := dbOnReceiver.GetKeycardsWithSameKeyUID(keycard1.KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(0, len(syncedKeycards))
}

func (s *MessengerSyncKeycardChangeSuite) TestDeleteKeycard() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	// Add keycard on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Add the same keycard on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Remove keycard from sender
	err = s.main.DeleteKeycard(context.Background(), keycard1.KeycardUID)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 1
		},
		"expected to receive keycard activities",
	)
	s.Require().NoError(err)

	senderKeycard, err := senderDb.GetKeycardByKeycardUID(keycard1.KeycardUID)
	s.Require().Error(err)
	s.Require().True(err == accounts.ErrNoKeycardForPassedKeycardUID)
	s.Require().Nil(senderKeycard)

	syncedKeycard, err := dbOnReceiver.GetKeycardByKeycardUID(keycard1.KeycardUID)
	s.Require().Error(err)
	s.Require().True(err == accounts.ErrNoKeycardForPassedKeycardUID)
	s.Require().Nil(syncedKeycard)
}

func (s *MessengerSyncKeycardChangeSuite) TestSettingKeycardName() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	// Add keycard on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Add the same keycard on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Prepare expected keycard for comparison
	updatedKeycard1 := accounts.GetProfileKeycardForTest()
	updatedKeycard1.KeycardName = "New Keycard Name"

	// Set new keycard name to sender
	err = s.main.SetKeycardName(context.Background(), updatedKeycard1.KeycardUID, updatedKeycard1.KeycardName)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 1
		},
		"expected to receive keycard activities",
	)
	s.Require().NoError(err)

	senderKeycard, err := senderDb.GetKeycardByKeycardUID(updatedKeycard1.KeycardUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeycards(updatedKeycard1, senderKeycard))

	syncedKeycard, err := dbOnReceiver.GetKeycardByKeycardUID(updatedKeycard1.KeycardUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeycards(updatedKeycard1, syncedKeycard))
}

func (s *MessengerSyncKeycardChangeSuite) TestSettingKeycardLocked() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	// Add keycard on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Add the same keycard on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Prepare expected keycard for comparison
	updatedKeycard1 := accounts.GetProfileKeycardForTest()
	updatedKeycard1.KeycardLocked = true

	err = s.main.KeycardLocked(context.Background(), updatedKeycard1.KeycardUID)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 1
		},
		"expected to receive keycard activities",
	)
	s.Require().NoError(err)

	senderKeycard, err := senderDb.GetKeycardByKeycardUID(updatedKeycard1.KeycardUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeycards(updatedKeycard1, senderKeycard))

	syncedKeycard, err := dbOnReceiver.GetKeycardByKeycardUID(updatedKeycard1.KeycardUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeycards(updatedKeycard1, syncedKeycard))
}

func (s *MessengerSyncKeycardChangeSuite) TestSettingKeycardUnlocked() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()
	keycard1.KeycardLocked = true

	// Add keycard on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Add the same keycard on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Prepare expected keycard for comparison
	updatedKeycard1 := accounts.GetProfileKeycardForTest()
	updatedKeycard1.KeycardLocked = false

	err = s.main.KeycardUnlocked(context.Background(), updatedKeycard1.KeycardUID)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 1
		},
		"expected to receive keycard activities",
	)
	s.Require().NoError(err)

	senderKeycard, err := senderDb.GetKeycardByKeycardUID(updatedKeycard1.KeycardUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeycards(updatedKeycard1, senderKeycard))

	syncedKeycard, err := dbOnReceiver.GetKeycardByKeycardUID(updatedKeycard1.KeycardUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeycards(updatedKeycard1, syncedKeycard))
}

func (s *MessengerSyncKeycardChangeSuite) TestUpdatingKeycardUid() {
	senderDb := s.main.settings
	dbOnReceiver := s.other.settings

	keycard1 := accounts.GetProfileKeycardForTest()

	// Add keycard on sender
	err := senderDb.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Add the same keycard on receiver
	err = dbOnReceiver.SaveOrUpdateKeycard(*keycard1, 0, false)
	s.Require().NoError(err)

	// Prepare expected keycard for comparison
	updatedKeycard1 := accounts.GetProfileKeycardForTest()
	updatedKeycard1.KeycardUID = "00000000000000000000000000000000"

	// Update keycard uid on sender
	err = s.main.UpdateKeycardUID(context.Background(), keycard1.KeycardUID, updatedKeycard1.KeycardUID)
	s.Require().NoError(err)

	// Wait for the response
	_, err = WaitOnMessengerResponse(
		s.other,
		func(r *MessengerResponse) bool {
			return len(r.Keypairs) == 1
		},
		"expected to receive keycard activities",
	)
	s.Require().NoError(err)

	senderKeycard, err := senderDb.GetKeycardByKeycardUID(updatedKeycard1.KeycardUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeycards(updatedKeycard1, senderKeycard))

	syncedKeycard, err := dbOnReceiver.GetKeycardByKeycardUID(updatedKeycard1.KeycardUID)
	s.Require().NoError(err)
	s.Require().True(accounts.SameKeycards(updatedKeycard1, syncedKeycard))
}
